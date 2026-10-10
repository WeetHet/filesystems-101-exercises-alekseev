//nolint:errorlint
package stack

import (
	"fmt"
	"io"
	"path"
	"runtime"
	"strconv"
	"strings"
)

// Frame represents a program counter inside a stack frame.
// For historical reasons if Frame is interpreted as a uintptr
// its value represents the program counter + 1.
type Frame uintptr

// pc returns the program counter for this frame;
// multiple frames may have the same PC value.
func (f Frame) pc() uintptr { return uintptr(f) - 1 }

// FileLine returns the full path to the file and line number that contains the
// function for this Frame's pc.
func (f Frame) FileLine() (string, int) {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown", 0 // nolint:goconst
	}
	return fn.FileLine(f.pc())
}

// Name returns the name of this function, if known.
func (f Frame) Name() string {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown"
	}
	return fn.Name()
}

// Format formats the frame according to the fmt.Formatter interface.
//
//	%s    source file
//	%d    source line
//	%n    function name
//	%v    equivalent to %s
//
// Format accepts flags that alter the printing of some verbs, as follows:
//
//	%+s   function name and path of source file relative to the compile time
//	      GOPATH as "<funcname> (<path>)"
//	%+v   equivalent to %+s
func (f Frame) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		file, line := f.FileLine()
		switch {
		case s.Flag('+'):
			_, _ = io.WriteString(s, f.Name())
			_, _ = io.WriteString(s, " (")
			_, _ = io.WriteString(s, file)
			_, _ = io.WriteString(s, ":")
			_, _ = io.WriteString(s, strconv.Itoa(line))
			_, _ = io.WriteString(s, ")")
		default:
			_, _ = io.WriteString(s, path.Base(file))
		}
	case 'd':
		_, line := f.FileLine()
		_, _ = io.WriteString(s, strconv.Itoa(line))
	case 'n':
		_, _ = io.WriteString(s, funcname(f.Name()))
	case 'v':
		f.Format(s, 's')
	}
}

// MarshalText formats a stacktrace Frame as a text string. The output is the
// same as that of fmt.Sprintf("%+v", f), but without newlines or tabs.
func (f Frame) MarshalText() ([]byte, error) {
	name := f.Name()
	if name == "unknown" {
		return []byte(name), nil
	}
	file, line := f.FileLine()
	return []byte(fmt.Sprintf("%s %s:%d", name, file, line)), nil
}

// funcname removes the path prefix component of a function's name reported by func.Name().
func funcname(name string) string {
	i := strings.LastIndex(name, "/")
	name = name[i+1:]
	i = strings.Index(name, ".")
	return name[i+1:]
}

// Trace is stack of Frames from innermost (newest) to outermost (oldest).
type Trace []Frame

// Format formats the stack of Frames according to the fmt.Formatter interface.
//
//	%s	lists source files for each Frame in the stack
//	%v	lists the source file and line number for each Frame in the stack
//
// Format accepts flags that alter the printing of some verbs, as follows:
//
//	%+v   Prints filename, function, and line number for each Frame in the stack.
func (st Trace) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			for idx, f := range st {
				f.Format(s, verb)
				if idx < len(st)-1 {
					_, _ = io.WriteString(s, ", ")
				}
			}
		case s.Flag('#'):
			fmt.Fprintf(s, "%#v", []Frame(st))
		default:
			st.formatSlice(s, verb)
		}
	case 's':
		st.formatSlice(s, verb)
	}
}

// formatSlice will format this StackTrace into the given buffer as a slice of
// Frame, only valid when called with '%s' or '%v'.
func (st Trace) formatSlice(s fmt.State, verb rune) {
	_, _ = io.WriteString(s, "[")
	for i, f := range st {
		if i > 0 {
			_, _ = io.WriteString(s, " ")
		}
		f.Format(s, verb)
	}
	_, _ = io.WriteString(s, "]")
}

// FormatAsStringArray will format stack trace as simple string array, w/o any special characters like \n.
// Strings are formatted as "func (file:line)"
func (st Trace) FormatAsStringArray() (res []string) {
	for _, frame := range st {
		file, line := frame.FileLine()
		s := frame.Name() + " (" + file + ":" + strconv.Itoa(line) + ")"
		res = append(res, s)
	}
	return
}

func (st Trace) isCalledFromInit() bool {
	if len(st) < 3 {
		return false
	}

	// call trace for init looks as:
	//
	// tests-runner/util/package_name.init
	// runtime.doInit
	// runtime.doInit   <-- we check this one
	// runtime.main
	// runtime.goexit
	//
	// NOTE: we don't check [0] element as one may have used incorrect "skip" while collecting call trace

	frame := st[len(st)-3]
	fn := runtime.FuncForPC(uintptr(frame))
	return fn != nil && fn.Name() == "runtime.doInit"
}

// Frames returns frames associated with StackTrace.
func (st Trace) Frames() []runtime.Frame {
	st0 := st.RawFrames()

	framesIter := runtime.CallersFrames(st0)
	frames := make([]runtime.Frame, 0)
	for {
		frame, more := framesIter.Next()
		frames = append(frames, frame)
		if !more {
			break
		}
	}

	return frames
}

// RawFrames convert StackTrace to []uintptr.
func (st Trace) RawFrames() []uintptr {
	// StackTrace is []Frame which essentially is []uintptr,
	// but go can't convert between those types.
	st0 := make([]uintptr, len(st))
	for i := range st {
		st0[i] = uintptr(st[i])
	}

	return st0
}

// Callers returns call trace frame pointers.
func Callers(skip int) Trace {
	const depth = 64
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])

	pcs0 := make([]Frame, n)
	for i := 0; i < n; i++ {
		pcs0[i] = Frame(pcs[i])
	}
	return pcs0
}

// CallersWithoutInit returns call trace frame pointers if not from init() function.
func CallersWithoutInit(skip int) Trace {
	st := Callers(skip + 1)
	if st.isCalledFromInit() {
		return nil
	}
	return st
}
