//nolint:errorlint
package errors

import (
	"context"
)

func init() {
	RegisterTranslator(contextErrorsTranslator)
}

// Translator modifies known error types to be returned by an API
type Translator interface {
	Translate() Error
}

// TranslateErrorHandler converts generic error to our structured errors
type TranslateErrorHandler = func(error) (ok bool, e Error)

var translateErrorHandlers []TranslateErrorHandler

// RegisterTranslator adds handler for error -> Error transformations, e.g. DB errors -> Error
func RegisterTranslator(h TranslateErrorHandler) {
	translateErrorHandlers = append(translateErrorHandlers, h)
}

// TranslateFromError returns structured error from original abstract one
func TranslateFromError(srcErr error) Error {
	if srcErr == nil {
		return Error{}
	}

	var translatableCause error
	var unwrapped = false
	var casted = false
	var translated Error
	curErr := srcErr
	for curErr != nil {
		e, ok := translateToErrorNoRecursion(curErr)
		if ok {
			_, casted = castToError(curErr)
			translatableCause = curErr
			translated = e
			break
		}
		curErr = Unwrap(curErr)
		unwrapped = true
	}

	// NOTE(eg): invalid Error (with empty Code) seems to
	//           be a non-sense, but lets protect from it
	//           just in case.
	if translatableCause != nil && translated.isValid() {
		// In case Error or *Error was found exactly at the top
		// then simply return translated error.
		if !unwrapped && casted {
			return translated
		}

		// otherwise we would like to keep original error
		// as a Reason so that stack traces and other possible
		// details are available in the returned error for
		// further investigation.
		translated.stack = nil
		return translated.WithReasonNoStack(srcErr)
	}

	// Last attempt is to use global translators.
	// NOTE: translators typically match through all errors in chain by using Is/As,
	// so they should not be called in main unwrap loop above.
	// Otherwise it can match before our Error in the chain is reached.
	for _, h := range translateErrorHandlers {
		if ok, e := h(srcErr); ok {
			return e.WithReasonNoStack(srcErr)
		}
	}

	return ErrInternal.WithReasonNoStack(srcErr)
}

// ErrCode returns our Code for any kind of given error (root cause), regardless if it's Error from this pkg or some abstract one
func ErrCode(err error) (code Code) {
	if err == nil {
		return
	}
	e := TranslateFromError(err)
	return e.Code
}

func translateToErrorNoRecursion(err error) (Error, bool) {
	if e, ok := castToError(err); ok {
		return e, true
	}

	if a, ok := err.(aser); ok {
		var e Error
		if a.As(&e) {
			return e, true
		}
	}

	if t, ok := err.(Translator); ok {
		return t.Translate(), true
	}

	return Error{}, false
}

func castToError(err error) (Error, bool) {
	if e, ok := err.(Error); ok {
		return e, true
	}
	if pe, ok := err.(*Error); ok {
		return *pe, true
	}
	return Error{}, false
}

func contextErrorsTranslator(err error) (ok bool, e Error) {
	if Is(err, context.Canceled) {
		return true, ErrCanceled
	}
	if Is(err, context.DeadlineExceeded) {
		return true, ErrDeadlineExceeded
	}
	return false, e
}
