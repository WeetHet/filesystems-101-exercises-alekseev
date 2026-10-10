package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"reflect"
	"sort"
	"strings"

	"github.com/gorilla/mux"

	"tests-runner/util/api"
	"tests-runner/util/errors"
	"tests-runner/util/gracefully"
	"tests-runner/util/iam"
	"tests-runner/util/logger"
	sctx "tests-runner/util/server/context"
	"tests-runner/util/server/internal/handler"
	"tests-runner/util/util/cond"
	"tests-runner/util/util/errutil"
)

const (
	// RouteDefault is implied when Route.Flags is not set
	RouteDefault = RouteAPIInternal
	// RouteMask selects bits defining route path
	RouteMask = 0xff

	// RouteAPIInternal registers route at /api/v1
	RouteAPIInternal = 1 << 0
	// RouteAPIAgent registers route at /api/agent
	RouteAPIAgent = 1 << 1
	// RouteWithoutPrefix registers route at / (no prefixes added)
	RouteWithoutPrefix = 1 << 7

	// RouteWithoutAuth disables authentication
	RouteWithoutAuth = 1 << 8
	// RouteWithoutACL disables ACL fetching
	RouteWithoutACL = 1 << 9
	// RouteWithoutTracing disables tracing
	RouteWithoutTracing = 1 << 10
	// RouteWithoutLogging disables request logging
	RouteWithoutLogging = 1 << 11
)

// Route describes API route and it's handler
type Route struct {
	Method     string // GET, POST, ...
	Path       string // path suffix (prepended by prefix)
	Name       string // human-readable route name for logging/tracing
	RawHandler func(w http.ResponseWriter, r *http.Request, acl *iam.ACL, params any) (err error)
	Handler    func(r *http.Request, acl *iam.ACL, params any) (result any, err error)
	Params     any // Params data type to be allocated/deserialized
	Result     any
	Flags      int
}

// DecodeParams returns decoded JSON/query params for route
func (r *Route) DecodeParams(req *http.Request) (params any, err error) {
	if r.Params == nil {
		return
	}

	t := reflect.TypeOf(r.Params)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	params = reflect.New(t).Interface()

	err = GetRequestParams(req, params)
	return
}

// GetPathPrefix returns a list of path prefixes to be used for the route
func (r *Route) GetPathPrefix() (prefix string) {
	fl := r.Flags & RouteMask
	fl = cond.If(fl == 0, RouteDefault, fl)
	errutil.BugOn(fl&(fl-1) != 0, "only one route flag should be set")

	if fl&RouteWithoutPrefix != 0 {
		prefix = ""
	}
	if fl&RouteAPIInternal != 0 {
		prefix = api.PrefixAPIInternal
	}
	if fl&RouteAPIAgent != 0 {
		prefix = api.PrefixAPIAgent
	}
	return prefix
}

func (s *Server) traceRoute(r Route, h http.Handler) http.Handler {
	return h
}

// AuthenticateAndInjectACL is a low level method to authenticate and inject ACL into
// the given context accordingly to the request data.
//
// WARN: Normally this method is not needed since almost all users should use Server.AddRoutes
// method to register routes which will take care of building the request context for handler
// with all the needed bits.
func (s *Server) AuthenticateAndInjectACL(ctx context.Context, req *http.Request) (context.Context, error) {
	return s.handlerContextBuilder.Build(ctx, handler.BuildContextParams{
		HTTPRequest:   req,
		ActorIP:       GetClientIP(req),
		Auth:          true,
		InjectACL:     true,
		NoDefaultTags: true,
	})
}

// AddRoute registers single route at given prefix path and authentication enabled/disabled
func (s *Server) AddRoute(r Route) {
	var h http.Handler
	h = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		ctx, err := s.handlerContextBuilder.Build(ctx, handler.BuildContextParams{
			HTTPRequest: req,
			ActorIP:     GetClientIP(req),
			Auth:        r.Flags&RouteWithoutAuth == 0,
			InjectACL:   r.Flags&RouteWithoutACL == 0,
		})
		// even in case of error the context might have
		// been updated with some useful values
		req = req.WithContext(ctx)
		if err != nil {
			s.HandleError(w, req, err)
			return
		}

		const maxContentLength = 32 * 1024 * 1024
		if r.Handler != nil || r.Params != nil { // r.RawHandler must take care on its own
			if req.ContentLength > maxContentLength {
				s.HandleError(w, req, errors.ErrOutOfRange.WithMessage("request body too large"))
				return
			}
			req.Body = http.MaxBytesReader(w, req.Body, maxContentLength)
		}

		params, err := r.DecodeParams(req)
		if err != nil {
			s.HandleError(w, req, err)
			return
		}

		if s.LogRequests && r.Flags&RouteWithoutLogging == 0 {
			s.logRequest(req, r.Name, params)
		}

		// ACL was added (or not) in handlerContextBuilder.Build called above
		acl, _ := sctx.ACL(ctx)

		if r.RawHandler != nil {
			// We use gracefully.Do here to sent panics to reporter, not to prevent panics!
			err = gracefully.Do(req.Context(), func(ctx context.Context) error {
				return r.RawHandler(w, req, acl, params)
			})

			// RawHandler can write raw bytes in HTTP response. If it wrote raw bytes in response, that means to web
			// was sent something like:
			//
			// HTTP 1/1
			// 200 OK
			// Content-Type: XXX
			//
			// <bytes>
			//
			// We can't write 400 Bad Request or 500 Internal Server Error on top. So the best thing what we can do
			// is to break connection with client. And that's what panic does
			// (which standard library is handled appropriately).
			//
			// NOTE: we use panic(http.ErrAbortHandler) to not write wrong stack trace to server's logs. While any
			// panic from ServeHTTP aborts the response to the client, panicking with ErrAbortHandler also
			// suppresses logging of a stack trace to the server's error log.
			//
			// NOTE: golang's stdlib uses `panic(ErrAbortHandler)` instead of `return ErrAbortHandler`.
			// Suppress the stack trace for such panics.
			if errors.Is(err, http.ErrAbortHandler) {
				// this may be a wrapped ErrAbortHandler or stdlib-style panic(ErrAbortHandler)
				panic(http.ErrAbortHandler)
			}
			if panicError := new(gracefully.PanicError); errors.As(err, &panicError) {
				s.Log.WithError(err).Errorf(req.Context(), "RawHandler panic: %+v", errors.GetStack(err))
				panic(http.ErrAbortHandler)
			}

			if err != nil {
				s.HandleError(w, req, err)
			}

			return
		}

		var res any
		err = gracefully.Do(req.Context(), func(ctx context.Context) error {
			res, err = r.Handler(req, acl, params)
			return err
		})
		switch {
		case err != nil:
			s.HandleError(w, req, err)
		default:
			s.WriteResponse(w, req, http.StatusOK, res)
		}
	})

	if r.Flags&RouteWithoutTracing == 0 {
		h = s.traceRoute(r, h)
	}
	path := path.Join(r.GetPathPrefix(), r.Path)
	s.routes = append(s.routes, r)
	s.Router().Methods(r.Method).Path(path).Handler(h)
}

// AddRoutes registers given routes in http mux
func (s *Server) AddRoutes(rts []Route) {
	for _, r := range rts {
		s.AddRoute(r)
	}
}

// GetRoutes returns a list of registered routes
func (s *Server) GetRoutes() (routes []Route) {
	return s.routes
}

func (s *Server) initCommonRoutes() {
	r := s.Router()
	if !s.DisableMonitoring {
		s.registerObservabilityEndpoints()
	}
	if !s.DisableCORS {
		r.PathPrefix("/").HandlerFunc(getCORSOptions).Methods(http.MethodOptions)
	}
}

func getCORSOptions(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, PUT, DELETE, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Acl-Hash, sentry-trace, baggage")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

type paramsLogFormatter interface {
	FormatForLog() string
}

func (s *Server) logRequest(req *http.Request, name string, params any) {
	if !s.Log.Enabled(logger.Debug) {
		return
	}

	ctx := req.Context()
	vars := mux.Vars(req)
	var buf strings.Builder
	buf.Grow(256)
	if name == "" {
		name = req.Method + " " + req.URL.Path
	}

	fmt.Fprintf(&buf, "HTTP request %q", name)
	if len(vars) > 0 {
		sortedVars := make([]string, 0, len(vars))
		for k, v := range vars {
			sortedVars = append(sortedVars, k+": "+v)
		}
		sort.Strings(sortedVars)
		fmt.Fprintf(&buf, ", vars: {%s}", strings.Join(sortedVars, ", "))
	}
	if params != nil {
		if fl, ok := params.(paramsLogFormatter); ok {
			fmt.Fprintf(&buf, ", params: {%s}", fl.FormatForLog())
		} else {
			data, err := json.Marshal(params)
			if err != nil {
				fmt.Fprintf(&buf, ", params: <failed to marshal>")
			} else {
				fmt.Fprintf(&buf, ", params: {%s}", string(data))
			}
		}
	}

	log := s.Log.WithField("request", name)
	if s.LogRequestsHeaders {
		log = log.WithField("request_headers", req.Header)
	}
	log.Debugf(ctx, "%s", buf.String())
}
