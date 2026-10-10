package server

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"tests-runner/util/errors"
)

// registerObservabilityEndpoints registers routes for health checks and debugging
func (s *Server) registerObservabilityEndpoints() {
	s.rMgmt.Path("/healthz").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := s.isHealthy(r.Context())
		if err != nil {
			s.HandleError(w, r, err)
			return
		}
		s.WriteResponse(w, r, http.StatusOK, nil)
	})

	s.rMgmt.Path("/ready").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := s.isReady(r.Context())
		if err != nil {
			s.HandleError(w, r, err)
			return
		}
		s.WriteResponse(w, r, http.StatusOK, nil)
	})

	s.rMgmt.Path("/metrics").Methods("GET").Handler(promhttp.Handler())
}

func (s *Server) isHealthy(ctx context.Context) error {
	if !s.isStarted() {
		return nil
	}
	return s.starter.IsHealthy(ctx)
}

func (s *Server) isReady(ctx context.Context) error {
	if !s.isStarted() {
		return errors.New("servers are not yet started")
	}
	if !s.starter.IsReady(ctx) {
		return errors.New("servers are not ready yet")
	}
	return nil
}

func (s *Server) isStarted() bool {
	s.RLock()
	defer s.RUnlock()
	return s.routesRegistered && s.starter.AllStarted()
}
