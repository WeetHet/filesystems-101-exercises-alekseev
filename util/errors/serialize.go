package errors

import (
	"encoding/json"
	"strings"
)

// any intermediate error implementing ErrorSerializable can be serialized/deserialized correctly as part of the chain
type ErrorSerializable interface {
	SerializeToChainError() ChainError
}

// helper type to make sure MarshalJSON is not called recursively
type errorWithoutMarshaller Error

// serverError is a type in which Error is serialized together with it's error chain
type serverError struct {
	*errorWithoutMarshaller
	Chain []ChainError `json:"chain,omitempty"`
}

type ChainErrorType string

const (
	BaseErrorType = ChainErrorType("base")
	CodeErrorType = ChainErrorType("code")
)

// ChainError defines a generic error description saved in chain array
type ChainError struct {
	Type    ChainErrorType `json:"type,omitempty"`
	Status  int            `json:"status,omitempty"`
	Code    Code           `json:"code,omitempty"`
	Message string         `json:"message"`
	Details details        `json:"details,omitempty"`
	Stack   []string       `json:"stack,omitempty"`

	reason error
}

// Unwrap implements errors.Unwrap() to make sure ChainError works correctly as std error
func (ce ChainError) Unwrap() error {
	return ce.reason
}

// Error returns error message
func (ce ChainError) Error() string {
	if ce.reason == nil {
		return ce.Message
	}
	return ce.Message + ": " + ce.reason.Error()
}

// SerializeToChainError returns itself to make sure ChainError is correctly serialized from chain
func (ce ChainError) SerializeToChainError() ChainError {
	return ce
}

// MarshalJSON serializes error chain to JSON array
func (e Error) MarshalJSON() ([]byte, error) {
	em := errorWithoutMarshaller(e)
	se := serverError{errorWithoutMarshaller: &em}

	for err := e.Reason; err != nil; err = Unwrap(err) {
		if chif, ok := err.(ErrorSerializable); ok { //nolint:errorlint
			se.Chain = append(se.Chain, chif.SerializeToChainError())
		} else {
			se.Chain = append(se.Chain, convertUnknownErrorToChain(err)...)
			break
		}
	}
	return json.Marshal(se)
}

// This function converts unknow 3rd party or core errors to a chain using Unwrap method.
// Unfortunately, it's not possible to get particular error message, so need to do string hacks.
func convertUnknownErrorToChain(err error) (ch []ChainError) {
	for xerr := err; xerr != nil; xerr = Unwrap(xerr) {
		ch = append(ch, ChainError{Type: BaseErrorType, Message: xerr.Error()})
	}

	// almost all Error() methods return "parent-error: child-error" message, so cut child error out of it.
	for i := 0; i < len(ch)-1; i++ {
		msg := strings.TrimSuffix(ch[i].Message, ch[i+1].Message)
		if len(msg) != len(ch[i].Message) {
			msg = strings.TrimRight(msg, ": ")
			ch[i].Message = msg
		}
	}

	return ch
}

// UnmarshalJSON deserializes error chain
func (e *Error) UnmarshalJSON(data []byte) (err error) {
	em := (*errorWithoutMarshaller)(e)
	se := serverError{errorWithoutMarshaller: em}

	if err = json.Unmarshal(data, &se); err != nil {
		return Wrap(err, "Failed to deserialize errors.Error")
	}

	if len(se.Chain) == 0 {
		return
	}
	var reason error
	for i := len(se.Chain) - 1; i >= 0; i-- {
		cerr := se.Chain[i]
		cerr.reason = reason
		err = cerr
		// this is a small hack, as we need errors of "Error" type to be restored correctly, e.g. for GetDetail to work.
		if cerr.Type == CodeErrorType {
			err = Error{
				HTTPStatus: cerr.Status,
				Code:       cerr.Code,
				Message:    cerr.Message,
				Details:    cerr.Details,
				Reason:     cerr.reason,
				stack:      foreignTrace(cerr.Stack),
			}
			if se.HTTPStatus == cerr.Status && se.Code == cerr.Code && se.Message == cerr.Message {
				// when we marshal error chain we take deep Error{} and put on top.
				// i.e. after unmarshal we have code Error twice in the chain.
				// it doesn't seem possible to fix here as it's method of Error{}
				se.dup = true
			}
		}
		reason = err
	}
	e.Reason = reason
	return nil
}

func (e Error) CleanAfterUnmarshal() error {
	if e.dup {
		// this error duplicates an inner error due to TranslateFromError() always makes
		// a top error to be Error{} by duplicating inner Error{}. So simply drop top error.
		return e.Reason
	}
	return e
}
