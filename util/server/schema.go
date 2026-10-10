package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"tests-runner/util/errors"
	"tests-runner/util/schema"
)

func isHttpTransportError(err error) bool {
	errMsg := err.Error()
	if strings.Contains(errMsg, "stream error: stream ID ") && (strings.Contains(errMsg, "INTERNAL_ERROR") || strings.Contains(errMsg, "CANCEL")) {
		return true
	}
	return false
}

// GetRequestJSONBody parses request JSON body to params
func GetRequestJSONBody(r *http.Request, params any) (err error) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		err = json.NewDecoder(r.Body).Decode(params)
	}
	if err != nil {
		msg := "Failed to decode the request body"
		xerr := errors.ErrInvalidArg.WithMessage(msg).WithReason(err)

		if errors.Is(err, io.EOF) {
			xerr = xerr.WithMessage(msg +
				". Ensure a valid, non-empty JSON body (e.g. 'null') is provided if the Content-Type='application/json' header is specified")
		}
		if isHttpTransportError(err) {
			xerr = xerr.WithSilenced() // don't spam to logs transport errors
		}
		err = xerr
	}
	return
}

// GetRequestParams fetches request params from JSON body and query arguments
func GetRequestParams(r *http.Request, params any) (err error) {
	// check body for JSON
	if err = GetRequestJSONBody(r, params); err != nil {
		return
	}

	err = schema.Decoder.Decode(params, r.URL.Query())
	if err != nil {
		err = errors.ErrInvalidArg.WithMessage("Failed to decode query parameters").Wrap(err)
	}
	return
}
