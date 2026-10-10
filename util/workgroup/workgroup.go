package workgroup

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"

	"tests-runner/util/errors"
	"tests-runner/util/gracefully"
	"tests-runner/util/util/errutil"
)

// Config describes workgroup parameters
type Config struct {
	// Concurrency of workgroup configures its concurrency level, i.e. number of goroutines:
	// >0 value is a maximum number of goroutines to spawn
	// zero value configures unbounded, i.e. maximum parallelism
	// -1 value is a special syncrhonous mode to avoid goroutines at all (and unwanted long call traces)
	Concurrency int
	// StopOnError can be set to fail all further goroutines from doing job if error encountered
	StopOnError bool
	// Graceful can be set to run all user workloads under gracefully.Do in order
	// to return regular wg.Error on panics.
	Graceful bool
}

// Wg runs task concurrently ensuring that not more than configured number of
// tasks are run simultaneously.
type Wg struct {
	cfg Config

	// sem limits the number of active worker goroutines
	sem        *semaphore.Weighted
	waitGroup  sync.WaitGroup
	terminated bool

	mtx          sync.Mutex // protect error state
	err          error      // first error occurred
	multipleErrs bool       // whether more than 1 goroutine exited with error
}

// WgFunc runs within WgFunc doing some job.
type WgFunc func(context.Context) error

// New returns a new instance of Wg which runs up to size tasks concurrently.
func New(cfg Config) *Wg {
	var sem *semaphore.Weighted
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 10000 // some sane limit
	}
	if cfg.Concurrency > 0 {
		sem = semaphore.NewWeighted(int64(cfg.Concurrency))
	}
	return &Wg{
		cfg: cfg,
		sem: sem,
	}
}

func (wg *Wg) do(ctx context.Context, stackErr *errors.BaseError, f WgFunc) {
	wg.mtx.Lock()
	err := wg.err // read under lock to make race detector happy
	wg.mtx.Unlock()

	if wg.cfg.StopOnError && err != nil {
		return
	}

	if wg.cfg.Graceful {
		err = gracefully.Do(ctx, f)
	} else {
		err = f(ctx)
	}
	if stackErr != nil {
		err = stackErr.Wrap(err)
	}
	wg.addError(err)
}

func (wg *Wg) addError(err error) {
	if err == nil {
		return
	}

	wg.mtx.Lock()
	if wg.err == nil {
		wg.err = err
	} else {
		wg.multipleErrs = true
	}
	wg.mtx.Unlock()
}

func (wg *Wg) acquireToken(ctx context.Context) error {
	if wg.sem != nil {
		return wg.sem.Acquire(ctx, 1)
	} else {
		return nil
	}
}

func (wg *Wg) releaseToken() {
	if wg.sem != nil {
		wg.sem.Release(1)
	}
}

// Go runs an f with no explicitly provided context.
func (wg *Wg) Go(f func() error) {
	wg.GoCtx(context.Background(), func(context.Context) error { return f() })
}

// GoCtx is same as Go() + additionally context is passed.
func (wg *Wg) GoCtx(ctx context.Context, f WgFunc) {
	if wg.cfg.Concurrency < 0 {
		wg.do(ctx, nil, f)
		return
	}

	if err := wg.acquireToken(ctx); err != nil {
		wg.addError(err)
		return
	}
	if wg.terminated {
		wg.addError(errors.New("waitgroup is terminated"))
		return
	}

	wg.waitGroup.Add(1)

	//nolint:errorlint
	stackErr := errors.New("workgroup task failed").(*errors.BaseError) // special error to capture stack trace

	go func() {
		// use defer to make sure it's done even if runtime.Goexit() is called
		defer wg.waitGroup.Done()
		defer wg.releaseToken()
		wg.do(ctx, stackErr, f)
	}()
}

// Wait for all workers to complete.
// Workgroup is fully stopped/terminated and can not be reused after Wait() is called.
// This is important property to make sure that Wait() is not racy vs. Go() when called from different contexts.
func (wg *Wg) Wait() {
	errutil.BugOn(wg.terminated, "workgroup can't be reused after Wait")
	// block spawning of new goroutines
	if wg.sem != nil {
		err := wg.sem.Acquire(context.Background(), int64(wg.cfg.Concurrency))
		errutil.BugOn(err != nil, "not possible")
	}

	wg.terminated = true
	wg.waitGroup.Wait()

	// release spawning of new goroutines to make then notice terminated flag
	if wg.sem != nil {
		wg.sem.Release(int64(wg.cfg.Concurrency))
	}
}

// Error returns the first error that occurred within Wg.
func (wg *Wg) Error() error {
	wg.mtx.Lock()
	defer wg.mtx.Unlock()

	err := wg.err
	if wg.multipleErrs {
		err = errors.Wrap(err, "multiple workgroup tasks failed")
	}

	return err
}
