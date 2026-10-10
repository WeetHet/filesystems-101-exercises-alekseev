package errutil

import (
	"fmt"
)

func recovered(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	f()
	return
}

// First returns first non-nil error out of errs, or nil.
func First(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// FatalIf panics if err is not nil.
func FatalIf(err error) {
	if err == nil {
		return
	}
	panic(fmt.Sprintf("FATAL: %v", err))
}
