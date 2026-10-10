package errors

import (
	"net/http"
)

const (
	// codes/implementation inspired by https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
	// additional comments: https://developers.google.com/maps-booking/reference/grpc-api/status_codes

	// CodeSuccess - no error
	CodeSuccess = Code("success")

	// CodeUnknown for all unclassified errors (aka Internal Server Error)
	// HTTP Mapping: 500 Internal Server Error
	CodeUnknown = Code("unknown")

	// CodeInternal means that some invariants expected by the
	// underlying system have been broken.  This error code is reserved
	// for serious errors.
	// HTTP Mapping: 500 Internal Server Error
	CodeInternal = Code("internal")

	// CodeInvalidArgument - some API call args are invalid. Details may have names of the arguments to be invalid
	// HTTP Mapping: 400 Bad Request
	CodeInvalidArgument = Code("invalid_argument")

	// CodeFailedPrecondition means:
	// The operation was rejected because the system is not in a state
	// required for the operation's execution.  For example, the directory
	// to be deleted is non-empty, an rmdir operation is applied to
	// a non-directory, etc.
	// HTTP Mapping: 412 Precondition failed
	CodeFailedPrecondition = Code("failed_precondition")

	// CodeOutOfRange means:
	// The operation was attempted past the valid range.  E.g., seeking or
	// reading past end-of-file.
	// HTTP Mapping: 400 Bad Request
	CodeOutOfRange = Code("out_of_range")

	// CodeUnsupported means:
	// The operation is not supported
	// HTTP Mapping: 400 Bad Request
	CodeUnsupported = Code("unsupported")

	// CodeTooManyRequests - Too many requests
	// HTTP Mapping: 429 Too Many Requests
	CodeTooManyRequests = Code("too_many_requests")

	// CodeDeadlineExceeded - Operation timed out
	// HTTP Mapping: 504 Gateway Timeout
	CodeDeadlineExceeded = Code("deadline_exceeded")

	// CodeCanceled - request was canceled by client
	// HTTP Mapping: 499 Client Closed Request
	CodeCanceled = Code("canceled")

	// CodeNotFound - object not found. Params typically names the object name not found
	// HTTP Mapping: 404 Not Found
	CodeNotFound = Code("not_found")

	// CodeGone - object no longer exists.
	// HTTP Mapping: 410 Gone
	CodeGone = Code("gone")

	// CodeConflict - there is some conflict and operation can't be completed
	// HTTP Mapping: 409 Conflict
	CodeConflict = Code("conflict")

	// CodeForbidden - not authorized to access the resource
	// HTTP Mapping: 403 Forbidden
	CodeForbidden = Code("forbidden")

	// CodeUnauthenticated - have no valid authentication credentials
	// HTTP Mapping: 401 Unauthorized
	CodeUnauthenticated = Code("unauthenticated")

	// CodeUnavailable - service unavailable
	// HTTP Mapping: 503 Service Unavailable
	CodeUnavailable = Code("unavailable")
)

// a set of predifined commonly used errors
var (
	ErrSuccess  = Error{HTTPStatus: http.StatusOK, Code: CodeSuccess, Message: "Operation completed successfully"}
	ErrUnknown  = Error{HTTPStatus: http.StatusInternalServerError, Code: CodeUnknown, Message: "Unknown error"}
	ErrInternal = Error{HTTPStatus: http.StatusInternalServerError, Code: CodeInternal, Message: "Internal error"}

	ErrInvalidArg         = Error{HTTPStatus: http.StatusBadRequest, Code: CodeInvalidArgument, Message: "Invalid argument"}
	ErrFailedPrecondition = Error{HTTPStatus: http.StatusPreconditionFailed, Code: CodeFailedPrecondition, Message: "Failed precondition"}
	ErrOutOfRange         = Error{HTTPStatus: http.StatusBadRequest, Code: CodeOutOfRange, Message: "Out of range"}
	ErrUnsupported        = Error{HTTPStatus: http.StatusBadRequest, Code: CodeUnsupported, Message: "Unsupported"}

	ErrTooManyRequests  = Error{HTTPStatus: http.StatusTooManyRequests, Code: CodeTooManyRequests, Message: "Too many requests in progress"}
	ErrDeadlineExceeded = Error{HTTPStatus: http.StatusGatewayTimeout, Code: CodeDeadlineExceeded, Message: "Deadline exceeded"}
	ErrCanceled         = Error{HTTPStatus: 499, Code: CodeCanceled, Message: "Canceled"}
	ErrUnavailable      = Error{HTTPStatus: http.StatusServiceUnavailable, Code: CodeUnavailable, Message: "Service unavailable"}

	ErrNotFound = Error{HTTPStatus: http.StatusNotFound, Code: CodeNotFound, Message: "Not found"}.WithSilenced()
	ErrGone     = Error{HTTPStatus: http.StatusGone, Code: CodeGone, Message: "Not found"}
	ErrConflict = Error{HTTPStatus: http.StatusConflict, Code: CodeConflict, Message: "Conflict"}

	ErrForbidden       = Error{HTTPStatus: http.StatusForbidden, Code: CodeForbidden, Message: "Forbidden"}
	ErrUnauthenticated = Error{HTTPStatus: http.StatusUnauthorized, Code: CodeUnauthenticated, Message: "Unauthenticated"}

	ErrAgentTerminated = Error{
		HTTPStatus: ErrCanceled.HTTPStatus,
		Code:       ErrCanceled.Code,
		Message:    "Operation suspended due to service termination",
	}.WithTaskDetails(&DetailTask{Resubmit: true})
)

// CodeToHTTPStatus converts our error code to http status
func CodeToHTTPStatus(c Code) int {
	switch c {
	case CodeCanceled:
		return 499
	case CodeConflict:
		return http.StatusConflict
	case CodeDeadlineExceeded:
		return http.StatusGatewayTimeout
	case CodeFailedPrecondition:
		return http.StatusPreconditionFailed
	case CodeForbidden:
		return http.StatusForbidden
	case CodeGone:
		return http.StatusGone
	case CodeInternal:
		return http.StatusInternalServerError
	case CodeInvalidArgument:
		return http.StatusBadRequest
	case CodeNotFound:
		return http.StatusNotFound
	case CodeOutOfRange:
		return http.StatusBadRequest
	case CodeSuccess:
		return http.StatusOK
	case CodeTooManyRequests:
		return http.StatusTooManyRequests
	case CodeUnauthenticated:
		return http.StatusUnauthorized
	case CodeUnknown:
		return http.StatusInternalServerError
	case CodeUnsupported:
		return http.StatusBadRequest
	case CodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
