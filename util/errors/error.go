//nolint:errorlint
package errors

import (
	"fmt"
	"io"
)

// Code represents short error codes
type Code string

// Error is an error to be returned from common api handlers
// Very similar to gRPC status errors: https://github.com/googleapis/googleapis/blob/master/google/rpc/status.proto
type Error struct {
	HTTPStatus int `json:"status"`
	// Code should be of the form: "{path.to.subsystem.short_error_description}".
	// For example:
	// - "archive.corrupted"
	// - "aad_api.unauthorized"
	// - "o365.api.cal.event_not_found"
	// It is OK to skip "path.to.subsystem" for common re-usable errors like:
	// - "not_found"
	// - "forbidden"
	Code    Code   `json:"code"`
	Message string `json:"message"`
	// Details examples: https://github.com/googleapis/googleapis/blob/master/google/rpc/error_details.proto
	Details details `json:"details,omitempty"`
	Reason  error   `json:"-"` // cause error in case of wrapping

	stack StackTrace // only deepest error carries call trace typically
	dup   bool       // means this error is a dup of inner chain error reachable by Unwrap()
}

// NewError returns an error representing code and msg.
func NewError(c Code, msg string) Error {
	if c == "" {
		c = CodeInternal
	}
	return Error{Code: c, Message: msg, stack: getStackTraceWithoutInit(3)}
}

// NewErrorf returns an errors with formatted msg
func NewErrorf(c Code, format string, a ...any) Error {
	return NewError(c, fmt.Sprintf(format, a...))
}

// StackTrace returns error stack trace if available
func (e Error) StackTrace() StackTrace { return e.stack }

// Error returns text message related to error
func (e Error) Error() string {
	var msg string
	if len(e.Details) == 0 {
		msg = e.Message + " (code=" + string(e.Code) + ")"
	} else {
		msg = fmt.Sprintf("%s (code=%s) (details=%s)", e.Message, e.Code, e.Details)
	}
	if e.Reason == nil {
		return msg
	}
	return msg + ": " + e.Reason.Error()
}

// Clone the error
func (e *Error) Clone() Error {
	c := *e
	if l := len(e.Details); l > 0 {
		c.Details = make(details, l)
		copy(c.Details, e.Details)
	}
	return c
}

// WithMessage returns cloned error with custom message
func (e Error) WithMessage(msg string) Error {
	ne := e.Clone()
	ne.Message = msg
	if ne.stack == nil {
		ne.stack = getStackTraceWithoutInit(3)
	}
	return ne
}

// WithCode returns cloned error with custom code
func (e Error) WithCode(code Code) Error {
	ne := e.Clone()
	ne.Code = code
	if ne.stack == nil {
		ne.stack = getStackTraceWithoutInit(3)
	}
	return ne
}

// WithStack returns cloned error with stack
func (e Error) WithStack() Error {
	ne := e.Clone()
	if ne.stack == nil {
		ne.stack = getStackTraceWithoutInit(3)
	}
	return ne
}

// WithReason returns cloned error with custom reason
func (e Error) WithReason(err error) Error {
	ne := e.Clone()
	ne.Reason = err
	if ne.stack == nil {
		ne.stack = getStackTraceWithoutInit(3)
	}
	return ne
}

// WithReasonNoStack returns cloned error with custom reason, but doesn't add stack trace
func (e Error) WithReasonNoStack(err error) Error {
	ne := e.Clone()
	ne.Reason = err
	return ne
}

// WithMessagef returns an error with formatted error
func (e Error) WithMessagef(msg string, args ...any) Error {
	ne := e.Clone()
	ne.Message = fmt.Sprintf(msg, args...)
	if ne.stack == nil {
		ne.stack = getStackTraceWithoutInit(3)
	}
	return ne
}

// Wrap returns cloned error with given error wrapped
// if e is a chain of errors, it attaches err to the last Reason
func (e Error) Wrap(err error) error {
	if err == nil {
		return nil
	}

	ne := e.Clone()
	if HasStack(err) {
		ne.stack = nil // no need to keep our stack if we have deeper one
	} else if ne.stack == nil {
		ne.stack = getStackTraceWithoutInit(3)
	}

	if ne.Reason == nil {
		ne.Reason = err
	} else {
		if er, ok := e.Reason.(interface{ Wrap(error) error }); ok {
			ne.Reason = er.Wrap(err)
		} else {
			panic("re-wrapping error not supporting Wrap()")
		}
	}

	return ne
}

// Unwrap provides compatibility with Go 1.13+ error chains
func (e Error) Unwrap() error {
	return e.Reason
}

// Format returns a formatted error, with stack trace if requested
func (e Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = io.WriteString(s, e.Error())

			// dump non-nil stack traces in reverse direction
			var stacks []StackTrace
			for xerr := error(e); xerr != nil; xerr = Unwrap(xerr) {
				stackTracer, ok := xerr.(StackTracer)
				if !ok {
					continue
				}
				st := stackTracer.StackTrace()
				if st == nil {
					continue
				}
				stacks = append(stacks, st)
			}
			if len(stacks) > 0 {
				_, _ = io.WriteString(s, " \n")
			}
			for i := len(stacks) - 1; i >= 0; i-- {
				stacks[i].Format(s, verb)
				if i > 0 {
					_, _ = io.WriteString(s, "\n~~~~~~~~~~~~~~~~~")
				}
			}
			return
		}
		fallthrough
	case 's', 'q':
		_, _ = io.WriteString(s, e.Error())
	}
}

// SerializeToChainError makes ChainError instead of Error to be marshaled in the middle of error chain
func (e Error) SerializeToChainError() ChainError {
	var st []string
	if e.stack != nil {
		st = e.stack.FormatAsStringArray()
	}
	return ChainError{
		Type:    CodeErrorType,
		Status:  e.HTTPStatus,
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
		Stack:   st,
	}
}

// Is consults Go1.13+ errors.Is
func (e Error) Is(target error) bool {
	if tErr, ok := target.(Error); ok {
		return e.Code == tErr.Code
	}
	if tErr, ok := target.(*Error); ok {
		return e.Code == tErr.Code
	}
	return false
}

func (e *Error) isValid() bool {
	return e.Code != ""
}
