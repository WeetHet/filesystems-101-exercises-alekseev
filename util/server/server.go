package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"tests-runner/util/errors"
	"tests-runner/util/logger"
	"tests-runner/util/server/internal/handler"
	"tests-runner/util/util/errutil"
	"tests-runner/util/version"
)

// DefaultReadHeaderTimeout is a default read header timeout per request, default is same as nginx value 60sec
const DefaultReadHeaderTimeout = 60 * time.Second

// DefaultIdleTimeout is a timeout that controls when the server closes inactive connections.
// Golang's stdlib relies on the server-side closing connections, so this must be configured
// on all http servers.
const DefaultIdleTimeout = 5 * time.Minute

// Config is compound server config
type Config struct {
	Name    string // executable name of service by default
	Version string // e.g. git tag or hash
	// BindAddr is an address used for HTTP server to listen on.
	BindAddr     string
	BindAddrMgmt string
	// Listener is a manually created net.Listener for HTTP server.
	// It is mutually exclusive with BindAddr.
	//
	// NOTE: http clients upgrade their connections to http/2 if the server accepts TLS
	// connections and includes "h2" in the TLSConfig. If you provide
	// a custom listener, it is your job to fill a TLSConfig the same way golang's http
	// server does.
	Listener net.Listener
	// DisableCORS prevent adding CORS header's and handler
	DisableCORS       bool
	DisableMonitoring bool
	// AllowH2C, if true, permits HTTP/2 requests using the insecure,
	// plain-text "http" scheme.
	AllowH2C bool

	Log logger.Logger

	AuthConfig
	DebugConfig
	HttpConfig *http.Server
}

func (cfg *Config) sanitize() error {
	if cfg.Name == "" {
		cfg.Name = path.Base(os.Args[0])
	}
	if cfg.Version == "" {
		cfg.Version = version.Version
	}
	if cfg.Log == nil {
		cfg.Log = logger.NewNoop()
	}

	if cfg.Listener != nil && cfg.BindAddr != "" {
		return errors.ErrInvalidArg.WithMessage("BindAddr and Listener are mutually exclusive")
	}
	if cfg.Listener != nil && cfg.BindAddrMgmt == "" && !cfg.DisableMonitoring {
		return errors.ErrInvalidArg.WithMessage("mgmt endpoints must be exposed only in the private network; servers that configure Listener must also configure BindAddrMgmt")
	}

	if cfg.HttpConfig == nil {
		cfg.HttpConfig = &http.Server{
			ReadHeaderTimeout: DefaultReadHeaderTimeout,
			IdleTimeout:       DefaultIdleTimeout,

			// http.Server uses ErrorLog mostly to notify of IO errors
			// like "TLS handshake failed". Silence them.
			ErrorLog: log.New(io.Discard, "", 0),
		}
	}
	if cfg.HttpConfig.ReadHeaderTimeout == 0 {
		return errors.ErrInvalidArg.WithMessage("http ReadHeaderTimeout must be configured")
	}
	if cfg.HttpConfig.IdleTimeout == 0 {
		return errors.ErrInvalidArg.WithMessage("http IdleTimeout must be configured")
	}
	if cfg.HttpConfig.Addr != "" {
		return errors.ErrInvalidArg.WithMessage("http Addr must be empty")
	}
	if cfg.HttpConfig.Handler != nil {
		return errors.ErrInvalidArg.WithMessage("http Handler must be empty")
	}
	if cfg.HttpConfig.TLSConfig != nil {
		return errors.ErrInvalidArg.WithMessage("http TLSConfig must be empty")
	}
	if cfg.HttpConfig.Protocols == nil {
		cfg.HttpConfig.Protocols = &http.Protocols{}
		cfg.HttpConfig.Protocols.SetHTTP1(true)
		cfg.HttpConfig.Protocols.SetHTTP2(true)
	}
	if cfg.AllowH2C {
		cfg.HttpConfig.Protocols.SetUnencryptedHTTP2(true)
	}

	return nil
}

// Server is compound HTTP server which unites and instruments subservers
// so that it is easy to create new services from scratch
type Server struct {
	Config

	sync.RWMutex
	// sub-components
	servers []SubServer
	routes  []Route
	// starter is responsible for starting/stopping subservers
	starter          *Starter
	routesRegistered bool
	// r is a router used to handle http subserver routes
	// and all utility routes (health checks, ...)
	r     *mux.Router
	rMgmt *mux.Router

	httpHandler http.Handler
	httpSrv     *http.Server

	handlerContextBuilder handler.ContextBuilder
}

// New creates Server instance from config and list of sub-servers (components):
// in case any of specified subservers are HTTPSubServer actually, when it is
// recognized and such subservers are used to register additional routes.
func New(ctx context.Context, cfg Config, subs ...SubServer) (s *Server, err error) {
	if err := cfg.sanitize(); err != nil {
		return nil, err
	}

	s = &Server{
		Config:  cfg,
		r:       mux.NewRouter(),
		servers: subs,
	}

	if s.BindAddrMgmt == "" {
		s.rMgmt = s.r
	} else {
		s.rMgmt = mux.NewRouter()
	}

	s.initCommonRoutes()

	if err = s.initAuth(ctx); err != nil {
		return
	}
	s.handlerContextBuilder = handler.ContextBuilder{
		AuthClient: s.AuthClient,
		ACLGetter:  s.ACLGetter,
	}

	s.httpHandler = s.r

	s.httpSrv = s.HttpConfig
	s.httpSrv.Handler = s

	return s, nil
}

// AddSubServer allow to add additional sub-servers before Start
func (s *Server) AddSubServer(sub ...SubServer) {
	errutil.BugOn(s.starter != nil, "subservers should be added before start")
	s.servers = append(s.servers, sub...)
}

// NewFromFlags creates Server instance from CLI flags
func NewFromFlags(ctx context.Context, log logger.Logger, f *Flags, subs ...SubServer) (s *Server, err error) {
	cfg := ConfigFromFlags(f)
	cfg.Log = log

	return New(ctx, cfg, subs...)
}

func (s *Server) registerRoutes() (err error) {
	s.Lock()
	defer s.Unlock()
	for _, ss := range s.servers {
		httpSS, ok := ss.(HTTPSubServer)
		if !ok {
			continue
		}

		if err = httpSS.AddRoutes(s); err != nil {
			return err
		}
	}
	s.routesRegistered = true
	return nil
}

// Start spawns all HTTP  and GRPC subservers.
// Routes are also registers for each HTTP subserver after start.
//
// In case of failure caller must invoke Stop() and Wait() to
// ensure that all started subservers are stopped.
//
// Also see ListenAndServe() method to avoid calling Start/Stop/Wait
// yourself.
func (s *Server) Start(ctx context.Context) (err error) {
	if err = s.startDebug(ctx); err != nil {
		return err
	}

	// start subservers
	s.starter = NewStarter(s.servers)
	if err = s.starter.Start(ctx); err != nil {
		return err
	}

	// subservers started => now we can expose their routes
	return s.registerRoutes()
}

// Stop all background activities
func (s *Server) Stop() {
	s.starter.Stop()
}

// Wait all background activities to exit
func (s *Server) Wait() {
	s.starter.Wait()
}

// Router can be used to add raw http handlers. Normally routes should be added via AddRoutes
func (s *Server) Router() *mux.Router {
	return s.r
}

// WriteDefaultHeaders issues default headers (e.g. CORS)
func (s *Server) WriteDefaultHeaders(w http.ResponseWriter) {
	// we can potentially host some static content besides API, so better to protect with STS
	w.Header().Set("Strict-Transport-Security", "max-age=63072000")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	// Browsers may cache some responses even with a JSON content type: we've
	// seen a 410 Gone response with a JSON body get cached. Successful GET
	// responses don't seem to be cached by modern browsers, but set no-store
	// explicitly so we don't rely on browser caching heuristics.
	w.Header().Set("Cache-Control", "no-store")
	if !s.DisableCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
}

// WriteResponse sends response with status + JSON
func (s *Server) WriteResponse(w http.ResponseWriter, req *http.Request, statusCode int, v any) {
	s.WriteDefaultHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(statusCode)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		s.Log.WithError(err).Errorf(req.Context(), "failed to encode response")
	}
}

// Redirect to specific url with statusCode
func (s *Server) Redirect(w http.ResponseWriter, r *http.Request, redirect ErrorRedirect) {
	s.WriteDefaultHeaders(w)
	http.Redirect(w, r, redirect.Target, redirect.HTTPStatus)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.RLock()
	defer s.RUnlock()
	s.httpHandler.ServeHTTP(w, r)
}

// Listen returns a net.Listener for HTTP server based on the provided configuration.
func (s *Server) Listen() (l net.Listener, err error) {
	if s.Listener != nil {
		return s.Listener, nil
	}

	l, err = net.Listen("tcp", s.BindAddr)
	if err != nil {
		return nil, err
	}

	return l, nil
}

// ListenAndServe starts compound server and waits for it to finish:
// - starts common http server
// - starts all dependent servers
// - blocks until ctx.Done or http server quit.
// - performs server.Stop() and Wait() for each server
func (s *Server) ListenAndServe(ctx context.Context) (err error) {
	log := s.Log

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Infof(ctx, "Server '%s' (ver %s/%s) starting...", s.Name, s.Version, runtime.Version())

	// start http listening and serving to expose health checks immediately
	l, err := s.Listen()
	if err != nil {
		return err
	}

	go func() {
		defer cancel()

		err := s.httpSrv.Serve(l)
		if err == nil {
			log.Infof(ctx, "http server quit")
		} else {
			log.WithError(err).Errorf(ctx, "http server shutdown error")
		}
	}()

	var mgmtSrv *http.Server
	if s.BindAddrMgmt != "" {
		mgmtSrv, err = s.StartMGMTServer(ctx, cancel)
		if err != nil {
			return err
		}
	}

	// start all subservers
	if err = s.Start(ctx); err != nil {
		log.WithError(err).Errorf(ctx, "Server start failed")
		goto stop
	}

	<-ctx.Done()
	err = ctx.Err()
stop:
	s.shutdownHttpServerGracefully(s.httpSrv)
	if mgmtSrv != nil {
		if err := mgmtSrv.Close(); err != nil {
			log.WithError(err).Errorf(ctx, "mgmtSrv.Close() failed")
		}
	}

	// stop and wait for subservers
	s.Stop()
	s.Wait()
	log.Infof(ctx, "Server %s terminated", s.Name)
	return err
}

func (s *Server) StartMGMTServer(ctx context.Context, cancel context.CancelFunc) (*http.Server, error) {
	var l net.Listener
	var err error
	log := s.Log
	l, err = net.Listen("tcp", s.BindAddrMgmt)
	if err != nil {
		return nil, err
	}

	mgmtSrv := &http.Server{
		Handler:           s.rMgmt,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		IdleTimeout:       DefaultIdleTimeout,
	}
	go func() {
		defer cancel()

		err := mgmtSrv.Serve(l)
		if err == nil {
			log.Infof(ctx, "mgmt http server quit")
		} else {
			log.WithError(err).Errorf(ctx, "mgmt http server shutdown error")
		}
	}()
	return mgmtSrv, nil
}

func (s *Server) shutdownHttpServerGracefully(httpServer *http.Server) {
	log := s.Log

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := httpServer.Shutdown(ctx); err != nil {
		log.WithError(err).Errorf(ctx, "httpServer.Shutdown() failed")
	}
	cancel()

	if err := httpServer.Close(); err != nil {
		log.WithError(err).Errorf(ctx, "httpServer.Close() failed")
	}
}
