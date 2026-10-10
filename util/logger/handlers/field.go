package handlers

import (
	"fmt"
	"math"
	"time"
	"unsafe"
)

func MkField(key string, value any) Field {
	return Field{key, anyValue(value)}
}

type Field struct {
	key   string
	value value
}

type value struct {
	_ [0]func() // disallow ==
	// num holds the value for Kinds Int64, Uint64, Float64, Bool and Duration,
	// the string length for KindString, and nanoseconds since the epoch for KindTime.
	num uint64
	// If any is of type Kind, then the value is in num as described above.
	// If any is of type *time.Location, then the Kind is Time and time.Time value
	// can be constructed from the Unix nanos in num and the location (monotonic time
	// is not preserved).
	// If any is of type stringptr, then the Kind is String and the string value
	// consists of the length in num and the pointer in any.
	// Otherwise, the Kind is Any and any is the value.
	// (This implies that Fields cannot store values of type Kind, *time.Location
	// or stringptr.)
	any any
}

type stringptr *byte // used in Value.any when the Value is a string

// Kind is the kind of a [value].
type Kind int

// The following list is sorted alphabetically, but it's also important that
// KindAny is 0 so that a zero Value represents nil.

const (
	KindAny Kind = iota
	KindBool
	KindDuration
	KindFloat64
	KindInt64
	KindString
	KindTime
	KindUint64
	KindGroup
	KindLogValuer
)

var kindStrings = []string{
	"Any",
	"Bool",
	"Duration",
	"Float64",
	"Int64",
	"String",
	"Time",
	"Uint64",
	"Group",
	"LogValuer",
}

func (k Kind) String() string {
	if k >= 0 && int(k) < len(kindStrings) {
		return kindStrings[k]
	} else {
		return fmt.Sprintf("<unknown logger.Kind %d>", int(k))
	}
}

// Unexported version of Kind, just so we can store Kinds in Values.
// (No user-provided value has this type.)
type kind Kind

// Kind returns v's Kind.
func (v value) Kind() Kind {
	switch x := v.any.(type) {
	case Kind:
		return x
	case stringptr:
		return KindString
	case timeLocation, timeTime:
		return KindTime
	case kind: // a kind is just a wrapper for a Kind
		return KindAny
	default:
		return KindAny
	}
}

type (
	// Unexported version of *time.Location, just so we can store *time.Locations in
	// Values. (No user-provided value has this type.)
	timeLocation *time.Location

	// timeTime is for times where UnixNano is undefined.
	timeTime time.Time
)

// Value constructors

func stringValue(v string) value {
	return value{num: uint64(len(v)), any: stringptr(unsafe.StringData(v))}
}

func int64Value(v int64) value {
	return value{num: uint64(v), any: KindInt64}
}

func uint64Value(v uint64) value {
	return value{num: v, any: KindUint64}
}

func float64Value(v float64) value {
	return value{num: math.Float64bits(v), any: KindFloat64}
}

func boolValue(v bool) value {
	if v {
		return value{num: 1, any: KindBool}
	} else {
		return value{num: 0, any: KindBool}
	}
}

func timeValue(v time.Time) value {
	if v.IsZero() {
		// UnixNano on the zero time is undefined, so represent the zero time
		// with a nil *time.Location instead. time.Time.Location method never
		// returns nil, so a Value with any == timeLocation(nil) cannot be
		// mistaken for any other Value, time.Time or otherwise.
		return value{any: timeLocation(nil)}
	}
	nsec := v.UnixNano()
	t := time.Unix(0, nsec)
	if v.Equal(t) {
		// UnixNano correctly represents the time, so use a zero-alloc representation.
		return value{num: uint64(nsec), any: timeLocation(v.Location())}
	}
	// Fall back to the general form.
	// Strip the monotonic portion to match the other representation.
	return value{any: timeTime(v.Round(0))}
}

func durationValue(v time.Duration) value {
	return value{num: uint64(v.Nanoseconds()), any: KindDuration}
}

func anyValue(v any) value {
	switch v := v.(type) {
	case string:
		return stringValue(v)

	case bool:
		return boolValue(v)

	case int:
		return int64Value(int64(v))
	case int8:
		return int64Value(int64(v))
	case int16:
		return int64Value(int64(v))
	case int32:
		return int64Value(int64(v))
	case int64:
		return int64Value(v)

	case uint:
		return uint64Value(uint64(v))
	case uint8:
		return uint64Value(uint64(v))
	case uint16:
		return uint64Value(uint64(v))
	case uint32:
		return uint64Value(uint64(v))
	case uintptr:
		return uint64Value(uint64(v))
	case uint64:
		return uint64Value(v)

	case float64:
		return float64Value(v)
	case float32:
		return float64Value(float64(v))

	case time.Duration:
		return durationValue(v)
	case time.Time:
		return timeValue(v)

	case Kind:
		return value{any: kind(v)}
	case value:
		return v
	default:
		return value{any: v}
	}
}

// Value accessors

func (v value) Any() any {
	switch v.Kind() {
	case KindAny:
		if k, ok := v.any.(kind); ok {
			return Kind(k)
		}
		return v.any
	case KindLogValuer:
		return v.any
	case KindInt64:
		return int64(v.num)
	case KindUint64:
		return v.num
	case KindFloat64:
		return v.Float64()
	case KindString:
		return v.String()
	case KindBool:
		return v.Bool()
	case KindDuration:
		return v.Duration()
	case KindTime:
		return v.Time()
	default:
		panic(fmt.Sprintf("bad kind %s", v.Kind()))
	}
}

func (v value) String() string {
	return unsafe.String(v.any.(stringptr), v.num)
}

func (v value) Int64() int64 {
	return int64(v.num)
}

func (v value) Uint64() uint64 {
	return v.num
}

func (v value) Bool() bool {
	return v.num == 1
}

func (v value) Duration() time.Duration {
	return time.Duration(int64(v.num))
}

func (v value) Float64() float64 {
	return math.Float64frombits(v.num)
}

func (v value) Time() time.Time {
	switch a := v.any.(type) {
	case timeLocation:
		if a == nil {
			return time.Time{}
		}
		return time.Unix(0, int64(v.num)).In(a)
	case timeTime:
		return time.Time(a)
	default:
		panic(fmt.Sprintf("bad time type %T", v.any))
	}
}
