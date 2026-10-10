package errors

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"tests-runner/util/util/errutil"
)

var (
	detailsTypeToString = make(map[reflect.Type]string)
	detailsStringToType = make(map[string]reflect.Type)
)

// Register detail type to make sure it survives marshal/unmarshal
func RegisterDetailType(t any, name string) {
	typ := reflect.TypeOf(t)

	if typ.Kind() != reflect.Pointer {
		panic("Only pointer types can be registered as details")
	}

	detailsTypeToString[typ] = name
	detailsStringToType[name] = typ
}

func getDetailType(d any) reflect.Type {
	// NOTE: this method does not panic in case `d` type is
	//       not registered yet using `RegisterDetailType` since
	//       this method can be used during package initializations
	//       before `RegisterDetailType` is called (for example inside
	//       stderr.go where standard errors are defined as package level
	//       global variables).
	t := reflect.TypeOf(d)
	if t.Kind() == reflect.Struct {
		return reflect.PointerTo(t)
	}
	return t
}

func init() {
	RegisterDetailType(&DetailInvalidArg{}, "invalid_arg")
	RegisterDetailType(&DetailRetryable{}, "retryable")
	RegisterDetailType(&DetailTask{}, "task")
	RegisterDetailType(&DetailSilenced{}, "silenced")
}

type detail struct {
	typ  reflect.Type
	data any
}

func (d *detail) MarshalJSON() ([]byte, error) {
	str, ok := detailsTypeToString[d.typ]
	if !ok {
		typStr := "nil"
		if d.typ != nil {
			typStr = d.typ.String()
		}
		panic(fmt.Sprintf("unsupported details type: %s", typStr))
	}

	data, err := json.Marshal(d.data)
	if err != nil {
		return nil, err
	}
	var dataMap map[string]any
	if err := json.Unmarshal(data, &dataMap); err != nil {
		return nil, err
	}
	dataMap["@type"] = str
	return json.Marshal(dataMap)
}

func (d *detail) UnmarshalJSON(data []byte) error {
	var t struct {
		Type string `json:"@type"`
	}
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	typ, ok := detailsStringToType[t.Type]
	if !ok {
		// Allow unknown types for backward compatibility
		return nil
	}

	val := reflect.New(typ.Elem()).Interface()
	if err := json.Unmarshal(data, val); err != nil {
		return err
	}

	d.typ = typ
	d.data = val
	return nil
}

type details []detail

// Custom UnmarshalJSON to filter out unsupported details
func (d *details) UnmarshalJSON(data []byte) error {
	type JSONdetails details
	var fulld JSONdetails

	if err := json.Unmarshal(data, &fulld); err != nil {
		return err
	}

	*d = nil
	for _, detail := range fulld {
		if detail.typ != nil {
			*d = append(*d, detail)
		}
	}

	return nil
}

func (d details) String() string {
	// Print details in JSON to have types in the output
	data, err := json.Marshal(d)
	if err != nil {
		return ""
	}
	return string(data)
}

// WithDetails returns cloned error with additional details
func (e Error) WithDetails(d any) Error {
	ne := e.Clone()
	if ne.stack == nil {
		ne.stack = getStackTraceWithoutInit(3)
	}

	typ := getDetailType(d)

	ne.Details = append(ne.Details, detail{
		typ:  typ,
		data: d,
	})
	return ne
}

// WithDetailsReplace returns cloned error with all details of given type replaced with d
func (e Error) WithDetailsReplace(d any) Error {
	ne := e.Clone()
	if ne.stack == nil {
		ne.stack = getStackTraceWithoutInit(3)
	}

	detail := detail{
		typ:  getDetailType(d),
		data: d,
	}

	// filter out all details of `d` type
	i := 0
	for _, ed := range ne.Details {
		if ed.typ != detail.typ {
			ne.Details[i] = ed
			i++
		}
	}
	// add detail to the end of the list as a replacement
	ne.Details = ne.Details[:i]
	ne.Details = append(ne.Details, detail)
	return ne
}

// HasDetail returns true is the error contains a detail of particular type
func (e Error) HasDetail(d any) bool {
	t := getDetailType(d)

	for i := range e.Details {
		if e.Details[i].typ == t {
			return true
		}
	}
	return false
}

// GetDetails returns a slice of details of particular type. If no such
// details exist, it returns an empty slice.
func (e Error) GetDetails(d any) (ret []any) {
	t := getDetailType(d)

	for _, detail := range e.Details {
		if detail.typ == t {
			ret = append(ret, detail.data)
		}
	}
	return
}

// GetDetail returns a very first error in the chain having a detail and detail itself
// if detailErr is nil no such error is found.
// detailErr can be used to get .Message corresponding to the error with the detail
func GetDetail[T any](err error, d T) (detail T, detailErr *Error) {
	if err == nil {
		return
	}

	t := getDetailType(d)

	for ; err != nil; err = Unwrap(err) {
		var e *Error
		if eptr, ok := err.(*Error); ok { // nolint:errorlint
			e = eptr
		}
		if eval, ok := err.(Error); ok { // nolint:errorlint
			e = &eval
		}
		if e == nil {
			continue
		}
		for _, detail := range e.Details {
			if detail.typ == t {
				return detail.data.(T), e
			}
		}
	}
	return
}

// ----------------------------------------------------------------------------------

// DetailInvalidArg describes violations in a client request
type DetailInvalidArg struct {
	Field       string `json:"field"`
	Description string `json:"description"`
}

// WithInvalidArg adds DetailInvalidArg with field violation details
func (e Error) WithInvalidArg(field, desc string, a ...any) Error {
	return e.WithDetails(&DetailInvalidArg{
		Field:       field,
		Description: fmt.Sprintf(desc, a...),
	})
}

// ----------------------------------------------------------------------------------

// DetailRetryable describes retryable in details
type DetailRetryable struct{}

// WithRetryable adds retryable detail to error
func (e Error) WithRetryable() Error {
	return e.WithDetailsReplace(&DetailRetryable{})
}

// IsRetryable return true if retryable detail set
func (e Error) IsRetryable() bool {
	return e.HasDetail(&DetailRetryable{})
}

// ----------------------------------------------------------------------------------

// DetailTask describes expected task reaction on the error
type DetailTask struct {
	// Ignore this error and complete task with OK status
	Ignore bool `json:"ignore,omitempty"`
	// may re-submit the task immediately, the code & message may be ignored then
	Resubmit bool `json:"resubmit,omitempty"`
	// may terminate the task with 'Canceled' status
	Cancel bool `json:"cancel,omitempty"`

	// Task may be suspended on SuspendDuration period and retried then
	Suspend         bool          `json:"suspend,omitempty"`
	SuspendDuration time.Duration `json:"suspend_duration,omitempty"`
}

// WithTaskDetails adds DetailTask with task details
func (e Error) WithTaskDetails(t *DetailTask) Error {
	errutil.BugOn(t.Resubmit && t.Suspend, "At most one of Resubmit and Suspend may be set.")
	return e.WithDetails(t)
}

// WithFatalTaskDetails adds DetailTask that task is fatal (not suspendable)
func (e Error) WithFatalTaskDetails() Error {
	return e.WithDetails(&DetailTask{Suspend: false})
}

func (e Error) GetTaskDetails() (dt *DetailTask) {
	dts := e.GetDetails(&DetailTask{})
	if len(dts) == 0 {
		return
	}
	return dts[0].(*DetailTask)
}

func GetTaskDetails(err error) (dt *DetailTask, derr *Error) {
	return GetDetail(err, dt)
}

// ----------------------------------------------------------------------------------

// DetailSilenced is a marker that this error mustn't be logged on the server side
type DetailSilenced struct{}

// WithSilenced adds a marker that this error must not be logged on the server side.
func (e Error) WithSilenced() Error {
	return e.WithDetailsReplace(&DetailSilenced{})
}

// IsSilenced reports whether a error must not be logged on the server side.
func (e Error) IsSilenced() bool {
	// NOTE: there is a difference between silenced and retryable details.
	// A error is retryable when the top-level error in the chain has a retryable detail.
	// A error is silenced when there is a error in the chain has a retryable detail.
	dt, _ := GetDetail(e, &DetailSilenced{})
	return dt != nil
}
