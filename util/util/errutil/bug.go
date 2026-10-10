package errutil

import (
	"fmt"
	_ "unsafe"
)

// Bug causes panic with formatted message msg.
//
//go:noescape
//go:linkname Bug
func Bug(format string, msg ...any)

// BugOn causes panic with formatted message msg if cond is true.
//
//go:noescape
//go:linkname BugOn
func BugOn(cond bool, format string, msg ...any)

// NOTE: this function is here to hide the body from the callers of Bug().
// A symbol Bug() is an alias to this function, and it is marked as noescape.
// That way the compiler can allocate the array of arguments on the stack,
// and it need not copy every argument to the heap.
//
//go:linkname _Bug tests-runner/util/util/errutil.Bug
func _Bug(format string, msg ...any) {
	panic(fmt.Sprintf("BUG: "+format, msg...))
}

//go:linkname _BugOn tests-runner/util/util/errutil.BugOn
func _BugOn(cond bool, format string, msg ...any) {
	if !cond {
		return
	}
	_Bug(format, msg...)
}
