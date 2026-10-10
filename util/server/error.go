package server

import (
	"net/http"

	"tests-runner/util/errors"
	"tests-runner/util/server/internal/handler"
)

// HandleError sends structured error
func (s *Server) HandleError(w http.ResponseWriter, req *http.Request, err error) {
	if ok, rerr := redirectFromError(err); ok {
		s.Redirect(w, req, rerr)
		return
	}
	serr := errors.TranslateFromError(err)
	s.LogReqError(req, serr)

	status := serr.HTTPStatus
	if status == 0 {
		s.Log.WithError(serr).Warnf(req.Context(), "Server returned Error with zero HTTPStatus")
		// Fallback to some for sure not retriable status code
		// to avoid "invalid WriteHeader code 0" panic. We use
		// 501 code here and not 500 because our clients will
		// retry 500.
		status = http.StatusNotImplemented
	}

	s.WriteResponse(w, req, status, serr)
}

// LogReqError logs structured error
func (s *Server) LogReqError(req *http.Request, err errors.Error) {
	handler.LogError(req.Context(), s.Log, handler.LogErrorParams{
		HTTPMethod: req.Method,
		HTTPPath:   req.URL.Path,
		Error:      err,
	})
}
