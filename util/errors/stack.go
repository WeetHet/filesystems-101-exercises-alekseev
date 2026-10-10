package errors

import (
	"fmt"
	"io"

	"tests-runner/util/stack"
)

// StackTracer is interface implemented by errors that have a stacktrace attached to them.
// NOTE: StackTrace() may return nil
type StackTracer interface {
	StackTrace() StackTrace
}

// StackTrace is an interface returned by errors.StackTrace(), i.e. errors capable to report its call traces.
// - stack.Trace from pkg/stack implements this interface for core traces.
// - foreignTrace from pkg/errors implements this interface for traces from other services.
type StackTrace interface {
	// At least FormatXXX methods should be supported by StackTrace interface
	Format(s fmt.State, verb rune)
	FormatAsStringArray() []string

	// RawFrames returns non-interpreted raw PCs.
	// It can be empty for other service errors (i.e. Unmarshalled), which still return Format()'ed trace yet.
	RawFrames() []uintptr
}

// -----------------------------------------------------------------------------------

// Implements stack.StackTrace for stacktraces of errors unserialised from JSONs.
type foreignTrace []string

func (tr foreignTrace) Format(s fmt.State, verb rune) {
	for i, line := range tr {
		_, _ = io.WriteString(s, line)
		if i < len(tr)-1 {
			_, _ = io.WriteString(s, ", ")
		}
	}
}

func (tr foreignTrace) FormatAsStringArray() []string { return tr }
func (tr foreignTrace) RawFrames() []uintptr          { return nil }

// -----------------------------------------------------------------------------------

// this function safely converts nil stack.Trace to StackTrace(nil) to avoid crashes due to nil checks
func getStackTraceWithoutInit(skip int) StackTrace {
	st := stack.CallersWithoutInit(skip + 1)
	if st == nil {
		return nil
	}
	return st
}

// HasStack verifies whether err has a stacktrace or not.
func HasStack(err error) bool {
	// optimisation to avoid unwrapping whole chain
	xerr, ok := err.(*BaseError) //nolint:errorlint
	if ok && xerr.hasStack {
		return true
	}

	return GetStack(err) != nil
}

// GetStack extracts first non-empty StackTrace() from the specified error chain.
func GetStack(err error) StackTrace {
	// Note, BaseError error implements interface but may return nil, so need to check the chain and can't use As(StackTracer)
	for xerr := err; xerr != nil; xerr = Unwrap(xerr) {
		if stackTracer, ok := xerr.(StackTracer); ok && stackTracer.StackTrace() != nil {
			return stackTracer.StackTrace()
		}
	}
	return nil
}
