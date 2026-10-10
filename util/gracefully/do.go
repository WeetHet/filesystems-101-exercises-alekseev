//nolint:errorlint
package gracefully

import (
	"context"

	"tests-runner/util/errors"
	"tests-runner/util/stack"
)

// Do executes f in a panic-protected environment.
// If a call to f() panics, Do() returns an instance of PanicError.
// If a Sentry reporter is configured for a process, an error will
// be reported there, too.
func Do(ctx context.Context, f func(ctx context.Context) error) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			// When panicking, go runtime does not unwind the stack. Instead, it runs
			// deferred functions as if they were called by the panic site. This way
			// runtime.Callers() inside a recover handler can learn the stack trace of
			// a function that panicked.
			//
			// Let us consider an example. Suppose f() calls g() which in turn calls h(),
			// and g() defers a function call to z(). When h() runs, the stack looks this
			// way:
			//     h()
			//     g()
			//     f()
			//     ...
			//     runtime.goexit()
			//
			// Now suppose that h() panics. Unlike an exception, this does not return
			// control to g(). Instead, a deferred call to z() is made atop this stack,
			// producing this:
			//     z()
			//     h()
			//     g()
			//     f()
			//     ...
			//     runtime.goexit()
			//
			// When a deferred call stops a panic with recover(), the go runtime pops
			// the stack frame of h(), then runs remaining deferred functions normally,
			// and then forces g() to return to f(). Control returns to f() even if g()
			// did not have a return at the place where it called h().
			//
			// For us, the important thing to learn from this description is that
			// runtime.Callers(3 + 1) removes the frame of a call deferred by
			// gracefully.Do(), which is this very function, and produces a stack
			// trace of a site that panicked.
			panicStack := stack.Callers(3 + 1)

			panicErr, ok := rec.(error)
			if !ok {
				panicErr = errors.Errorf("%v", rec)
			} else {
				panicErr = errors.Wrap(panicErr, "panic")
			}
			panicErr = panicErr.(*errors.BaseError).WithStack(panicStack)

			// It looks like recognize panics errors by checking that msg field contains "panic" is not the best
			// approach, so we are creating a new special kind of error.
			err = &PanicError{BaseError: *panicErr.(*errors.BaseError)}
		}
	}()

	return f(ctx)
}

// DoFn returns a function that executes `f` in a panic-protected environment.
func DoFn(ctx context.Context, f func(ctx context.Context) error) func() error {
	return func() error {
		return Do(ctx, f)
	}
}

// PanicError is an error that Do() uses to wrap a panic inside a handler.
type PanicError struct {
	errors.BaseError
}

// SerializeToChainError makes ChainError instead of BaseError to be marshaled in the middle of error chain
func (e *PanicError) SerializeToChainError() errors.ChainError {
	ce := e.BaseError.SerializeToChainError()
	ce.Type = "panic"
	return ce
}
