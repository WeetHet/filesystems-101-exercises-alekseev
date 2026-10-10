package schema

import (
	"reflect"
	"time"

	"github.com/gorilla/schema"

	"tests-runner/util/ulid"
	"tests-runner/util/util/cond"
)

// We add schema support for our own data types like ULID here
var Encoder *schema.Encoder
var Decoder *schema.Decoder

func init() {
	// NOTE: schema encodes nil ptrs as "null" and is not able to decode it later itself.
	// so don't forget to add 'omitempty' to all optional fields with pointers.

	Encoder = schema.NewEncoder()
	Encoder.RegisterEncoder(ulid.ULID{}, func(v reflect.Value) string {
		id := v.Interface().(ulid.ULID)
		return cond.If(id == ulid.Nil, "", id.String())
	})
	Encoder.RegisterEncoder(time.Time{}, func(v reflect.Value) string {
		t := v.Interface().(time.Time)
		return cond.If(t.IsZero(), "", t.Format(time.RFC3339))
	})
	Encoder.RegisterEncoder(&time.Time{}, func(v reflect.Value) string {
		t := v.Interface().(*time.Time)
		if t == nil {
			return ""
		}
		return cond.If(t.IsZero(), "", t.Format(time.RFC3339))
	})

	// --------------------------------------------------------------------------

	Decoder = schema.NewDecoder()

	// ignore unknown fields (behave similar to encoding/json)
	Decoder.IgnoreUnknownKeys(true)

	Decoder.RegisterConverter(ulid.ULID{}, func(value string) (v reflect.Value) {
		// gorilla/schema is very funny: it can detect and use UnmarshalText
		// method to parse ulid if it is just a standalone ulid.ULID in the struct.
		// But, if we keep []ulid.ULID in the struct then gorilla/schema returns
		// error. However, if we RegisterConverter for ulid.ULID it starts to work,
		// but, it actually does not use this converter still =) It is simply the
		// check that converter exists fixes it. But to actually decode slice elements
		// UnmarshalText is still used by gorilla =)

		if value == "" {
			return reflect.ValueOf(ulid.Nil)
		}

		id, err := ulid.Parse(value)
		if err != nil {
			return v // v.IsValid false
		}
		return reflect.ValueOf(id)
	})
	Decoder.RegisterConverter(time.Time{}, func(value string) (v reflect.Value) {
		if value == "" {
			return reflect.ValueOf(time.Time{})
		}

		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return v // v.IsValid false
		}
		return reflect.ValueOf(t)
	})
}
