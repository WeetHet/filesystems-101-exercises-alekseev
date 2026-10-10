//nolint:errorlint
package ctxutil

import (
	"context"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"sync/atomic"

	"tests-runner/util/errors"
)

// CancelReasonFunc is like stdlib context.CancelFunc but
// it also communicates concrete reason the context was
// canceled due. In case nil error specified then standard
// context.Canceled will be used.
type CancelReasonFunc func(err error)

// WithCancelReason is like stdlib context.WithCancel but with
// additional ability to cancel resulting context with a specific
// error reason.
//
// Might be deleted in case golang team decides to accept proposal:
//
//	https://github.com/golang/go/issues/46273
//
// NOTE: our version returns unwrappable 'canceled' error, while mainstream - not... :/
func WithCancelReason(parent context.Context) (context.Context, CancelReasonFunc) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	c := newCancelCtx(parent)
	propagateCancel(parent, c)
	return c, func(err error) {
		if err == nil {
			err = &cancelationError{
				BaseError: *errors.New("context canceled without error").(*errors.BaseError),
			}
		} else {
			err = &cancelationError{
				BaseError: *errors.Wrap(err, "context canceled").(*errors.BaseError),
			}
		}
		c.cancel(true, err)
	}
}

type cancelationError struct {
	errors.BaseError
}

func (e *cancelationError) Is(err error) bool {
	return err == context.Canceled
}

// PropagateCancel creates context based upon both parent and child which
// is guaranteed to be canceled once
// context is cancelled. This utility method might be helpful when you have
// a global context which might be cancelled and once it is then all ongoing
// operations should be canceled too.
//
// Call cancel once done working with resulting context to release resources.
func PropagateCancel(parent context.Context, child context.Context) (context.Context, CancelReasonFunc) {
	if parent == nil {
		panic("cannot propagate cancel from nil parent")
	}
	cancelableChild, cancel := WithCancelReason(child)
	propagateCancel(parent, cancelableChild.(canceler))
	return cancelableChild, func(err error) {
		if err == nil {
			err = context.Canceled
		}
		cancel(err)
	}
}

// =========================================================================
// ========== copy from stdlib go/src/os/signal/signal.go ==================
// =========================================================================

// NotifyContext returns a copy of the parent context that is marked done
// (its Done channel is closed) when one of the listed signals arrives,
// when the returned stop function is called with the specified error signalErr,
// or when the parent context's Done channel is closed, whichever happens first.
//
// The stop function unregisters the signal behavior, which, like signal.Reset,
// may restore the default behavior for a given signal. For example, the default
// behavior of a Go program receiving os.Interrupt is to exit. Calling
// NotifyContext(parent, os.Interrupt) will change the behavior to cancel
// the returned context. Future interrupts received will not trigger the default
// (exit) behavior until the returned stop function is called.
//
// The stop function releases resources associated with it, so code should
// call stop as soon as the operations running in this Context complete and
// signals no longer need to be diverted to the context.
func NotifyContext(parent context.Context, signalErr error, signals ...os.Signal) (ctx context.Context, stop CancelReasonFunc) {
	ctx, cancel := WithCancelReason(parent)
	c := &signalCtx{
		Context: ctx,
		cancel:  cancel,
		signals: signals,
	}
	c.ch = make(chan os.Signal, 1)
	signal.Notify(c.ch, c.signals...)
	if ctx.Err() == nil {
		go func() {
			select {
			case <-c.ch:
				c.cancel(signalErr)
			case <-c.Done():
			}
		}()
	}
	return c, c.stop
}

type signalCtx struct {
	context.Context

	cancel  CancelReasonFunc
	signals []os.Signal
	ch      chan os.Signal
}

func (c *signalCtx) stop(err error) {
	c.cancel(err)
	signal.Stop(c.ch)
}

func (c *signalCtx) String() string {
	var buf []byte
	// We know that the type of c.Context is cancelCtx, and we know that the
	// String method of cancelCtx returns a string that ends with ".WithCancel".
	name := c.Context.(stringer).String()
	name = name[:len(name)-len(".WithCancel")]
	buf = append(buf, "NotifyContext("+name...)
	if len(c.signals) != 0 {
		buf = append(buf, ", ["...)
		for i, s := range c.signals {
			buf = append(buf, s.String()...)
			if i != len(c.signals)-1 {
				buf = append(buf, ' ')
			}
		}
		buf = append(buf, ']')
	}
	buf = append(buf, ')')
	return string(buf)
}

// =========================================================================
// ========== copy from stdlib go/srv/context/context.go ===================
// =========================================================================

// newCancelCtx returns an initialized cancelCtx.
func newCancelCtx(parent context.Context) *cancelCtx {
	return &cancelCtx{Context: parent}
}

// goroutines counts the number of goroutines ever created; for testing.
var goroutines int32

// propagateCancel arranges for child to be canceled when parent is.
func propagateCancel(parent context.Context, child canceler) {
	done := parent.Done()
	if done == nil {
		return // parent is never canceled
	}

	select {
	case <-done:
		// parent is already canceled
		child.cancel(false, parent.Err())
		return
	default:
	}

	if p, ok := parentCancelCtx(parent); ok {
		p.mu.Lock()
		if p.err != nil {
			// parent has already been canceled
			child.cancel(false, p.err)
		} else {
			if p.children == nil {
				p.children = make(map[canceler]struct{})
			}
			p.children[child] = struct{}{}
		}
		p.mu.Unlock()
	} else {
		atomic.AddInt32(&goroutines, +1)
		go func() {
			select {
			case <-parent.Done():
				child.cancel(false, parent.Err())
			case <-child.Done():
			}
		}()
	}
}

// &cancelCtxKey is the key that a cancelCtx returns itself for.
var cancelCtxKey int

// parentCancelCtx returns the underlying *cancelCtx for parent.
// It does this by looking up parent.Value(&cancelCtxKey) to find
// the innermost enclosing *cancelCtx and then checking whether
// parent.Done() matches that *cancelCtx. (If not, the *cancelCtx
// has been wrapped in a custom implementation providing a
// different done channel, in which case we should not bypass it.)
func parentCancelCtx(parent context.Context) (*cancelCtx, bool) {
	done := parent.Done()
	if done == closedchan || done == nil {
		return nil, false
	}
	p, ok := parent.Value(&cancelCtxKey).(*cancelCtx)
	if !ok {
		return nil, false
	}
	pdone, _ := p.done.Load().(chan struct{})
	if pdone != done {
		return nil, false
	}
	return p, true
}

// removeChild removes a context from its parent.
func removeChild(parent context.Context, child canceler) {
	p, ok := parentCancelCtx(parent)
	if !ok {
		return
	}
	p.mu.Lock()
	if p.children != nil {
		delete(p.children, child)
	}
	p.mu.Unlock()
}

// A canceler is a context type that can be canceled directly. The
// implementations are *cancelCtx and *timerCtx.
type canceler interface {
	cancel(removeFromParent bool, err error)
	Done() <-chan struct{}
}

// closedchan is a reusable closed channel.
var closedchan = make(chan struct{})

func init() {
	close(closedchan)
}

// A cancelCtx can be canceled. When canceled, it also cancels any children
// that implement canceler.
type cancelCtx struct {
	context.Context

	mu       sync.Mutex            // protects following fields
	done     atomic.Value          // of chan struct{}, created lazily, closed by first cancel call
	children map[canceler]struct{} // set to nil by the first cancel call
	err      error                 // set to non-nil by the first cancel call
}

func (c *cancelCtx) Value(key any) any {
	if key == &cancelCtxKey {
		return c
	}
	return c.Context.Value(key)
}

func (c *cancelCtx) Done() <-chan struct{} {
	d := c.done.Load()
	if d != nil {
		return d.(chan struct{})
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	d = c.done.Load()
	if d == nil {
		d = make(chan struct{})
		c.done.Store(d)
	}
	return d.(chan struct{})
}

func (c *cancelCtx) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}

type stringer interface {
	String() string
}

func contextName(c context.Context) string {
	if s, ok := c.(stringer); ok {
		return s.String()
	}
	return reflect.TypeOf(c).String()
}

func (c *cancelCtx) String() string {
	return contextName(c.Context) + ".WithCancel"
}

// cancel closes c.done, cancels each of c's children, and, if
// removeFromParent is true, removes c from its parent's children.
func (c *cancelCtx) cancel(removeFromParent bool, err error) {
	if err == nil {
		panic("context: internal error: missing cancel error")
	}
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return // already canceled
	}
	c.err = err
	d, _ := c.done.Load().(chan struct{})
	if d == nil {
		c.done.Store(closedchan)
	} else {
		close(d)
	}
	for child := range c.children {
		// NOTE: acquiring the child's lock while holding parent's lock.
		child.cancel(false, err)
	}
	c.children = nil
	c.mu.Unlock()

	if removeFromParent {
		removeChild(c.Context, c)
	}
}

// WithValuesFromContext builds a context which Deadline(), Err(), Done() are
// taken from the first ctx argument, but values returned by Value(key) method
// are joined from both of the contexts with values from valuesCtx prioritized
// over values from ctx.
func WithValuesFromContext(ctx context.Context, valuesCtx context.Context) context.Context {
	return withValuesFromCtx{Context: ctx, valuesCtx: valuesCtx}
}

type withValuesFromCtx struct {
	context.Context
	valuesCtx context.Context
}

func (c withValuesFromCtx) Value(key any) any {
	v := c.valuesCtx.Value(key)
	if v != nil {
		return v
	}
	return c.Context.Value(key)
}
