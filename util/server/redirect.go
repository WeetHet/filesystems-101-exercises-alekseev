package server

import (
	"fmt"
	"net/http"

	"tests-runner/util/errors"
)

var (
	// ErrRedirect from handler results in REDIRECT reply
	ErrRedirect = ErrorRedirect{HTTPStatus: http.StatusMovedPermanently}
)

// ErrorRedirect is an error to be returned from common api handlers for perform redirect
type ErrorRedirect struct {
	HTTPStatus int    `json:"status"`
	Target     string `json:"target"`
}

// Error returns text message related to error
func (e ErrorRedirect) Error() string {
	return fmt.Sprintf("srv redirect httpStatus=%d target=%s", e.HTTPStatus, e.Target)
}

// WithTarget returns cloned error with given target
func (e ErrorRedirect) WithTarget(target string) ErrorRedirect {
	return ErrorRedirect{HTTPStatus: e.HTTPStatus, Target: target}
}

// redirectFromError retrieves ErrorRedirect from generic errors. Returns nil on fail
func redirectFromError(err error) (ok bool, redir ErrorRedirect) {
	if ok = errors.As(err, &redir); ok {
		return
	}

	return
}
