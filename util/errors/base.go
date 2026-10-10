// Package errors provides simple error handling primitives.
// Based on github.com/pkg/errors
//
// The traditional error handling idiom in Go is roughly akin to
//
//	if err != nil {
//	        return err
//	}
//
// which when applied recursively up the call stack results in error reports
// without context or debugging information. The errors package allows
// programmers to add context to the failure path in their code in a way
// that does not destroy the original value of the error.
//
// # Adding context to an error
//
// The errors.Wrap function returns a new error that adds context to the
// original error by recording a stack trace at the point Wrap is called,
// together with the supplied message. For example
//
//	_, err := ioutil.ReadAll(r)
//	if err != nil {
//	        return errors.Wrap(err, "read failed")
//	}
//
// # Retrieving the cause of an error
//
// Using errors.Wrap constructs a stack of errors, adding context to the
// preceding error. Depending on the nature of the error it may be necessary
// to reverse the operation of errors.Wrap to retrieve the original error
// for inspection. Errors can be "unwrapped" usign std Go errors package interface:
//
//	type unwrapper interface {
//	        Unwrap() error
//	}
//
// # Formatted printing of errors
//
// All error values returned from this package implement fmt.Formatter and can
// be formatted by the fmt package. The following verbs are supported:
//
//	%s    print the error. If the error has a Reason it will be
//	      printed recursively.
//	%v    see %s
//	%+v   extended format. Each Frame of the error's StackTrace will
//	      be printed in detail.
//
// # Retrieving the stack trace of an error or wrapper
//
// New, Errorf, Wrap, and Wrapf record a stack trace at the point they are
// invoked if the inner error doesn't have stack trace yet.
// This information can be retrieved with the following interface:
//
//	type StackTracer interface {
//	        StackTrace() errors.StackTrace
//	}
//
// The returned errors.StackTrace type is defined as
//
//	type StackTrace []Frame
//
// The Frame type represents a call site in the stack trace. Frame supports
// the fmt.Formatter interface that can be used for printing information about
// the stack trace of this error. For example:
//
//	if err, ok := err.(StackTracer); ok {
//	        for _, f := range err.StackTrace() {
//	                fmt.Printf("%+s:%d\n", f, f)
//	        }
//	}
//
// See the documentation for Frame.Format for more details.
//
//nolint:errorlint
package errors

import (
	stderrors "errors"
	"fmt"
	"io"
	"reflect"
)

// BaseError is an error that has a message and optionally a stack and/or wrapped error.
type BaseError struct {
	msg string
	err error

	stack    StackTrace // only deepest error carries call trace typically
	hasStack bool       // indicates if this error or inner error has stack
}

// New returns an error with the supplied message.
// New also records the stack trace at the point it was called.
func New(message string) error {
	st := getStackTraceWithoutInit(3)
	return &BaseError{
		msg:      message,
		stack:    st,
		hasStack: st != nil,
	}
}

// Errorf formats according to a format specifier and returns the string
// as a value that satisfies error.
// Errorf also records the stack trace at the point it was called.
func Errorf(format string, args ...any) error {
	st := getStackTraceWithoutInit(3)
	return &BaseError{
		msg:      fmt.Sprintf(format, args...),
		stack:    st,
		hasStack: st != nil,
	}
}

// Wrap creates a chain of errors (wrapped)
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}

	var st StackTrace
	if !HasStack(err) {
		st = getStackTraceWithoutInit(3)
	}

	return &BaseError{
		msg:      message,
		err:      err,
		stack:    st,
		hasStack: true,
	}
}

// Wrapf creates a chain of errors with formatted message for top error
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}

	st := StackTrace(nil)
	if !HasStack(err) {
		st = getStackTraceWithoutInit(3)
	}

	return &BaseError{
		msg:      fmt.Sprintf(format, args...),
		err:      err,
		stack:    st,
		hasStack: true,
	}
}

// WrapfWithStack creates a chain of errors with a message and stack trace attached to the error,
// even if inner error already has a stack. This allows to glue multiple stack traces from goroutines like that:
//
//	go {
//	  ...
//	  errCh <- errors.New()    // <--- stack captured here
//	}
//	err := <- errCh
//	return errors.WrapfWithStack(err, "Failed to execute")   // <--- and stack captured here
func WrapfWithStack(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}

	return &BaseError{
		msg:      fmt.Sprintf(format, args...),
		err:      err,
		stack:    getStackTraceWithoutInit(3),
		hasStack: true,
	}
}

// WithStack sets a custom stack trace to this error.
func (e *BaseError) WithStack(st StackTrace) *BaseError {
	return &BaseError{
		msg:      e.msg,
		err:      e.err,
		stack:    st,
		hasStack: true,
	}
}

// Unwrap returns std errors.Unwrap result
func Unwrap(err error) error { return stderrors.Unwrap(err) }

// Wrap returns cloned error with given error wrapped
// if e is a chain of errors, it attaches err to the last in chain
func (e *BaseError) Wrap(err error) error {
	if err == nil {
		return nil
	}

	ne := *e // clone

	if ne.err == nil {
		ne.err = err
	} else {
		if er, ok := e.err.(interface{ Wrap(error) error }); ok {
			ne.err = er.Wrap(err)
		} else {
			panic("re-wrapping error not supporting Wrap()")
		}
	}

	return &ne
}

// Unwrap returns inner chain error
func (e *BaseError) Unwrap() error { return e.err }

// StackTrace returns error stack trace if available or nil if not (inner error is likely to have in this case)
func (e *BaseError) StackTrace() StackTrace { return e.stack }

// Error returns a simple error message with all wrapped errors chain
func (e *BaseError) Error() string {
	if e.err == nil {
		return e.msg
	}
	return e.msg + ": " + e.err.Error()
}

// Format returns a formatted error, with stack trace if requested
func (e *BaseError) Format(s fmt.State, verb rune) {
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

// SerializeToChainError makes ChainError instead of BaseError to be marshaled in the middle of error chain
func (e *BaseError) SerializeToChainError() ChainError {
	var st []string
	if e.stack != nil {
		st = e.stack.FormatAsStringArray()
	}
	return ChainError{
		Type:    BaseErrorType,
		Message: e.msg,
		Stack:   st,
	}
}

// ------------------------------------------------------------------------------------------------------------------

type unwrapper interface {
	Unwrap() error
}

type iser interface {
	Is(error) bool
}

type aser interface {
	As(any) bool
}

var (
	errorType     = reflect.TypeFor[error]()
	unwrapperType = reflect.TypeFor[unwrapper]()
	iserType      = reflect.TypeFor[iser]()
	aserType      = reflect.TypeFor[aser]()
)

// dereferenceError tries to dereference 'err' until error interface and, if true for original value,
// Unwrap / As methods are implemented
func dereferenceError(err error) error {
	if err == nil {
		return nil
	}
	for {
		errVal := reflect.ValueOf(err)
		if errVal.Kind() != reflect.Pointer || errVal.IsNil() {
			break
		}
		deRef := errVal.Elem()
		if !deRef.Type().Implements(errorType) {
			break
		}
		_, errImplementsUnwrap := err.(unwrapper)
		if errImplementsUnwrap && !deRef.Type().Implements(unwrapperType) {
			// example: withStack / withMessage from github.com/pkg/errors implement error as values,
			// but Unwrap() require pointer
			break
		}
		_, errImplementsAs := err.(aser)
		if errImplementsAs && !deRef.Type().Implements(aserType) {
			break
		}
		_, errImplementsIs := err.(iser)
		if errImplementsIs && !deRef.Type().Implements(iserType) {
			break
		}
		err = deRef.Interface().(error)
	}
	return err
}

// Is is similar to standard library errors.Is, but a bit more convenient.
//
// Unfortunately, if some type Error implements standard error interface,
// pointer *Error implements this interface too, but standard errors.Is,
// weird enough, will return expected result for Error only, not for *Error.
// Therefore, it is needed to dereference the pointer to error.
//
// Also standard library errors.Is cannot reliably check if target if comparable
// and it can panic. See: https://github.com/golang/go/issues/54709
//
// See TestIs for examples and differences from standard library.
func Is(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}

	isComparable := isComparable(target)
	err = dereferenceError(err)
	for {
		if x, ok := err.(iser); ok {
			if x.Is(target) {
				return true
			}
		}
		if isComparable && err == target {
			return true
		}

		// Unwrap
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			err = dereferenceError(err) // FYI: additional action vs stderrors.Is()
			if err == nil {
				return false
			}
		case interface{ Unwrap() []error }:
			for _, err := range x.Unwrap() {
				if Is(err, target) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
}

func isComparable(x any) bool {
	if x == nil {
		return true
	}
	return isComparableValue(reflect.ValueOf(x))
}

// We should check comparability of the variables very carefully, since Type.Comparable() from stdlib reflect package
// returns true for interfaces even if there will be a run-time panic on actual comparison.
// https://go.dev/ref/spec#Comparison_operators
func isComparableValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice, reflect.Map, reflect.Func:
		// Slice, map, and function values are not comparable
		return false
	case reflect.Array:
		// Array values are comparable if values of the array element type are comparable.
		for i, n := 0, v.Len(); i < n; i++ {
			if !isComparableValue(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Interface:
		// Interface values are comparable. <...>
		// A comparison of two interface values with identical dynamic types causes a run-time panic
		// if values of that type are not comparable.
		return isComparableValue(v.Elem())
	case reflect.Struct:
		// Struct values are comparable if all their fields are comparable.
		for i, n := 0, v.NumField(); i < n; i++ {
			if !isComparableValue(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		// Bool, Int, Int8, Int16, Int32, Int64, Uint, Uint8, Uint16, Uint32, Uint64, Uintptr,
		// Float32, Float64, Complex64, Complex128, Chan, Pointer, String, UnsafePointer
		return true
	}
}

// As is similar to standard library errors.As, but a bit more convenient.
//
// Unfortunately, if some type Error implements standard error interface,
// pointer *Error implements this interface too, but standard errors.As,
// weird enough, will return expected result for Error only, not for *Error.
// Therefore it is needed to dereference the pointer to error.
//
// See TestAs for examples and differences from standard library.
func As(err error, target any) bool {
	if err == nil {
		return false // slight optimization
	}

	if target == nil {
		panic("errors: target cannot be nil")
	}
	targetVal := reflect.ValueOf(target)
	targetTyp := targetVal.Type()
	if targetTyp.Kind() != reflect.Pointer || targetVal.IsNil() {
		panic("errors: target must be a non-nil pointer")
	}

	deRefTargetType := targetTyp.Elem()
	if deRefTargetType.Kind() != reflect.Interface && !deRefTargetType.Implements(errorType) {
		panic("errors: *target must be interface or implement error")
	}

	err = dereferenceError(err)

	for err != nil {
		if reflect.TypeOf(err).AssignableTo(deRefTargetType) {
			targetVal.Elem().Set(reflect.ValueOf(err))
			return true
		}
		if x, ok := err.(aser); ok && x.As(target) {
			return true
		}

		// Unwrap
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			err = dereferenceError(err) // FYI: additional action vs stderrors.As()
			if err == nil {
				return false
			}
		case interface{ Unwrap() []error }:
			for _, err := range x.Unwrap() {
				if As(err, target) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
	return false
}
