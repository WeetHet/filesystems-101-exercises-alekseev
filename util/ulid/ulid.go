// Copyright 2016 The Oklog Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ulid

import (
	"bytes"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"tests-runner/util/errors"
)

/*
An ULID is a 16 byte Universally Unique Lexicographically Sortable Identifier

	The components are encoded as 16 octets.
	Each component is encoded with the MSB first (network byte order).

	0                   1                   2                   3
	0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|                      32_bit_uint_time_high                    |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|     16_bit_uint_time_low      |       16_bit_uint_random      |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|                       32_bit_uint_random                      |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|                       32_bit_uint_random                      |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/
type ULID [16]byte

// Nil is special form of ULID that is specified to have all 128 bits set to
// zero.
var Nil ULID

var (
	// ErrInvalidCharacters is returned when parsing or unmarshaling ULIDs with
	// invalid Base32 encodings.
	ErrInvalidCharacters error = errors.ErrInvalidArg.WithMessage("ulid: bad data characters when unmarshaling")

	// ErrBigTime is returned when constructing an ULID with a time that is larger
	// than MaxTime.
	ErrBigTime error = errors.ErrInvalidArg.WithMessage("ulid: time too big")

	// ErrInvalidSize is returned when unmarshaling a ULID that is not 26 chars long.
	ErrInvalidSize error = errors.ErrInvalidArg.WithMessage("ulid: bad data size")

	// ErrOverflow is returned when unmarshaling a ULID whose first character is
	// larger than 7, thereby exceeding the valid bit depth of 128.
	ErrOverflow error = errors.ErrInvalidArg.WithMessage("ulid: overflow when unmarshaling")
)

// safeRand wraps math/rand.Rand for concurrent use
type safeRand struct {
	mu   sync.Mutex
	rand *rand.Rand
}

// Read implements io.Reader interface
func (r *safeRand) Read(b []byte) (int, error) {
	r.mu.Lock()
	n, err := r.rand.Read(b) // nolint:staticcheck
	r.mu.Unlock()

	return n, err
}

var entropy = &safeRand{
	// nolint:gosec
	// G404: Use of weak random number generator (math/rand instead of crypto/rand)
	rand: rand.New(rand.NewSource(time.Now().UnixNano())),
}

// New is a no arguments version, using default entropy and current time
// Default current time can be overridden for tests through pkg/util/timeutil func SetNowFuncOverride.
func New() ULID {
	return MustNew(NowMs(), entropy)
}

// NewV4 is provided for better compatibility with uuid pkg
func NewV4() ULID {
	return New()
}

// New2 returns an ULID with the given Unix milliseconds timestamp and an
// optional entropy source. Use the Timestamp function to convert
// a time.Time to Unix milliseconds.
//
// ErrBigTime is returned when passing a timestamp bigger than MaxTime.
// Reading from the entropy source may also return an error.
func New2(ms uint64, entropy io.Reader) (id ULID, err error) {
	if err = id.SetTimeMs(ms); err != nil {
		return id, err
	}

	if entropy != nil {
		_, err = entropy.Read(id[6:])
	}

	return id, err
}

// MustNew is a convenience function equivalent to New that panics on failure
// instead of returning an error.
func MustNew(ms uint64, entropy io.Reader) ULID {
	id, err := New2(ms, entropy)
	if err != nil {
		panic(err)
	}
	return id
}

// Parse parses an encoded ULID, returning an error in case of failure.
//
// ErrDataSize is returned if the len(ulid) is different from an encoded
// ULID's length. Invalid encodings produce undefined ULIDs. For a version that
// returns an error instead, see ParseStrict.
func Parse(ulid string) (id ULID, err error) {
	return id, parse([]byte(ulid), &id)
}

// FromString is an alias to Parse() for better compibility with uuid pkg
func FromString(ulid string) (id ULID, err error) {
	return Parse(ulid)
}

// Merge generates new unique (repeatable) ulid from 2 existing ones with max time part and XOR'ed entropy part
func Merge(ulid1, ulid2 ULID) (id ULID) {
	t1, t2 := ulid1.TimeMs(), ulid2.TimeMs()
	if t1 > t2 {
		copy(id[:6], ulid1[:6])
	} else {
		copy(id[:6], ulid2[:6])
	}
	// XOR entropy of both
	copy(id[6:], ulid1[6:])
	for i := 6; i < 16; i++ {
		id[i] ^= ulid2[i]
	}
	return
}

// MergeHashed merges two ULID identifiers:
//   - ULID timestamp part is set to `ts`
//   - ULID entropy part is set to first 10 bytes of sha sum of concatenated entropy
//     parts of ulid1 and ulid2
func MergeHashed(ulid1, ulid2 ULID) (id ULID) {
	ms := max(ulid1.TimeMs(), ulid2.TimeMs())

	sha := sha256.Sum224(append(ulid1.Entropy(), ulid2.Entropy()...))
	id, err := New2(ms, bytes.NewReader(sha[:]))
	if err != nil {
		panic("impossible")
	}
	return id
}

func parse(v []byte, id *ULID) error {
	// Check if a base32 encoded ULID is the right length.
	if len(v) != EncodedSize {
		return ErrInvalidSize
	}

	// Check if the first character in a base32 encoded ULID will overflow. This
	// happens because the base32 representation encodes 130 bits, while the
	// ULID is only 128 bits.
	//
	// See https://github.com/oklog/ulid/issues/9 for details.
	if v[0] > '7' {
		return ErrOverflow
	}

	var (
		x0, x1, x2, x3, x4, x5, x6 uint32
		acc, bad                   uint32
	)

	x0 = uint32(dec[v[0+0]])
	x1 = uint32(dec[v[0+1]])
	x2 = uint32(dec[v[0+2]])
	x3 = uint32(dec[v[0+3]])
	x4 = uint32(dec[v[0+4]])
	x5 = uint32(dec[v[0+5]])
	x6 = uint32(dec[v[0+6]])

	bad = (x0 | x1 | x2 | x3 | x4 | x5 | x6) &^ 0x1F
	if bad != 0 {
		return ErrInvalidCharacters
	}

	acc |= (x0 << 29) | (x1 << 24) | (x2 << 19) | (x3 << 14) |
		(x4 << 9) | (x5 << 4) | (x6 >> 1)
	binary.BigEndian.PutUint32(id[0:4], acc)
	acc = (x6 & 0x1) << 31

	x0 = uint32(dec[v[7+0]])
	x1 = uint32(dec[v[7+1]])
	x2 = uint32(dec[v[7+2]])
	x3 = uint32(dec[v[7+3]])
	x4 = uint32(dec[v[7+4]])
	x5 = uint32(dec[v[7+5]])
	x6 = uint32(dec[v[7+6]])

	bad = (x0 | x1 | x2 | x3 | x4 | x5 | x6) &^ 0x1F
	if bad != 0 {
		return ErrInvalidCharacters
	}

	acc |= (x0 << 26) | (x1 << 21) | (x2 << 16) | (x3 << 11) |
		(x4 << 6) | (x5 << 1) | (x6 >> 4)
	binary.BigEndian.PutUint32(id[4:8], acc)
	acc = (x6 & 0xF) << 28

	x0 = uint32(dec[v[14+0]])
	x1 = uint32(dec[v[14+1]])
	x2 = uint32(dec[v[14+2]])
	x3 = uint32(dec[v[14+3]])
	x4 = uint32(dec[v[14+4]])
	x5 = uint32(dec[v[14+5]])

	bad = (x0 | x1 | x2 | x3 | x4 | x5) &^ 0x1F
	if bad != 0 {
		return ErrInvalidCharacters
	}

	acc |= (x0 << 23) | (x1 << 18) | (x2 << 13) | (x3 << 8) |
		(x4 << 3) | (x5 >> 2)
	binary.BigEndian.PutUint32(id[8:12], acc)
	acc = (x5 & 0x7) << 30

	x0 = uint32(dec[v[20+0]])
	x1 = uint32(dec[v[20+1]])
	x2 = uint32(dec[v[20+2]])
	x3 = uint32(dec[v[20+3]])
	x4 = uint32(dec[v[20+4]])
	x5 = uint32(dec[v[20+5]])

	bad = (x0 | x1 | x2 | x3 | x4 | x5) &^ 0x1F
	if bad != 0 {
		return ErrInvalidCharacters
	}

	acc |= (x0 << 25) | (x1 << 20) | (x2 << 15) | (x3 << 10) |
		(x4 << 5) | (x5 << 0)
	binary.BigEndian.PutUint32(id[12:16], acc)

	return nil
}

// MustParse is a convenience function equivalent to Parse that panics on failure
// instead of returning an error.
func MustParse(ulid string) (id ULID) {
	err := parse([]byte(ulid), &id)
	if err != nil {
		panic("an ULID must be 26 alphanumeric symbols except I, L, O, U")
	}
	return id
}

// String returns a lexicographically sortable string encoded ULID
// (26 characters, non-standard base 32) e.g. 01AN4Z07BY79KA1307SR9X4MV3
// Format: tttttttttteeeeeeeeeeeeeeee where t is time and e is entropy
func (id ULID) String() string {
	ulid := make([]byte, EncodedSize)
	_ = id.MarshalTextTo(ulid)
	return string(ulid)
}

// Format implementation controls how type rune are interpreted to generate its output.
func (id ULID) Format(f fmt.State, c rune) {
	_, _ = io.WriteString(f, id.String())
}

// Bytes returns bytes slice representation of UUID.
func (id ULID) Bytes() []byte {
	return id[:]
}

// MarshalBinary implements the encoding.BinaryMarshaler interface by
// returning the ULID as a byte slice.
func (id ULID) MarshalBinary() ([]byte, error) {
	ulid := make([]byte, len(id))
	return ulid, id.MarshalBinaryTo(ulid)
}

// MarshalBinaryTo writes the binary encoding of the ULID to the given buffer.
func (id ULID) MarshalBinaryTo(dst []byte) error {
	if len(dst) != len(id) {
		return fmt.Errorf("ulid: bad buffer size (%v) in MarshalBinaryTo", len(dst))
	}

	copy(dst, id[:])
	return nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface by
// copying the passed data and converting it to an ULID. ErrDataSize is
// returned if the data length is different from ULID length.
func (id *ULID) UnmarshalBinary(data []byte) error {
	if len(data) != len(*id) {
		return fmt.Errorf("ulid: bad data size (%v) in UnmarshalBinary", len(data))
	}

	copy((*id)[:], data)
	return nil
}

// MarshalText implements the encoding.TextMarshaler interface by
// returning the string encoded ULID.
func (id ULID) MarshalText() ([]byte, error) {
	ulid := make([]byte, EncodedSize)
	return ulid, id.MarshalTextTo(ulid)
}

// MarshalTextTo writes the ULID as a string to the given buffer.
func (id ULID) MarshalTextTo(dst []byte) error {
	// Optimized unrolled loop ahead.
	// From https://github.com/RobThree/NUlid

	if len(dst) != EncodedSize {
		return ErrInvalidSize
	}

	// 10 byte timestamp
	x0 := id[0]
	x1 := id[1]
	x2 := id[2]
	x3 := id[3]
	x4 := id[4]
	x5 := id[5]
	dst[0] = enc[(x0&224)>>5]
	dst[1] = enc[x0&31]
	dst[2] = enc[(x1&248)>>3]
	dst[3] = enc[((x1&7)<<2)|((x2&192)>>6)]
	dst[4] = enc[(x2&62)>>1]
	dst[5] = enc[((x2&1)<<4)|((x3&240)>>4)]
	dst[6] = enc[((x3&15)<<1)|((x4&128)>>7)]
	dst[7] = enc[(x4&124)>>2]
	dst[8] = enc[((x4&3)<<3)|((x5&224)>>5)]
	dst[9] = enc[x5&31]

	// 16 bytes of entropy
	x6 := id[6]
	x7 := id[7]
	x8 := id[8]
	x9 := id[9]
	x10 := id[10]
	x11 := id[11]
	x12 := id[12]
	x13 := id[13]
	x14 := id[14]
	x15 := id[15]
	dst[10] = enc[(x6&248)>>3]
	dst[11] = enc[((x6&7)<<2)|((x7&192)>>6)]
	dst[12] = enc[(x7&62)>>1]
	dst[13] = enc[((x7&1)<<4)|((x8&240)>>4)]
	dst[14] = enc[((x8&15)<<1)|((x9&128)>>7)]
	dst[15] = enc[(x9&124)>>2]
	dst[16] = enc[((x9&3)<<3)|((x10&224)>>5)]
	dst[17] = enc[x10&31]
	dst[18] = enc[(x11&248)>>3]
	dst[19] = enc[((x11&7)<<2)|((x12&192)>>6)]
	dst[20] = enc[(x12&62)>>1]
	dst[21] = enc[((x12&1)<<4)|((x13&240)>>4)]
	dst[22] = enc[((x13&15)<<1)|((x14&128)>>7)]
	dst[23] = enc[(x14&124)>>2]
	dst[24] = enc[((x14&3)<<3)|((x15&224)>>5)]
	dst[25] = enc[x15&31]

	return nil
}

// Byte to index table for O(1) lookups when unmarshaling.
// We use 0xFF as sentinel value for invalid indexes.
var dec = [...]byte{
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x01,
	0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E,
	0x0F, 0x10, 0x11, 0xFF, 0x12, 0x13, 0xFF, 0x14, 0x15, 0xFF,
	0x16, 0x17, 0x18, 0x19, 0x1A, 0xFF, 0x1B, 0x1C, 0x1D, 0x1E,
	0x1F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x0A, 0x0B, 0x0C,
	0x0D, 0x0E, 0x0F, 0x10, 0x11, 0xFF, 0x12, 0x13, 0xFF, 0x14,
	0x15, 0xFF, 0x16, 0x17, 0x18, 0x19, 0x1A, 0xFF, 0x1B, 0x1C,
	0x1D, 0x1E, 0x1F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
}

var enc = [...]byte{
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F',
	'G', 'H', 'J', 'K', 'M', 'N', 'P', 'Q', 'R', 'S', 'T', 'V', 'W', 'X', 'Y', 'Z',
	'?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?',
	'?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?',
	'?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?',
	'?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?',
	'?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?',
	'?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?',
	'?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?',
	'?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?',
	'?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?',
	'?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?',
	'?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?',
	'?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?',
	'?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?',
	'?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?', '?',
}

// EncodedSize is the length of a text encoded ULID.
const EncodedSize = 26

// UnmarshalText implements the encoding.TextUnmarshaler interface by
// parsing the data as string encoded ULID.
//
// This function validates that v is a valid string representing an ULID.
func (id *ULID) UnmarshalText(v []byte) error {
	return parse(v, id)
}

// MarshalJSON implements json.Marshaler interface.
func (id ULID) MarshalJSON() ([]byte, error) {
	// NOTE: the compiler can't prove that ulid.Nil is always zero.
	if id == (ULID{}) {
		return []byte("null"), nil
	}

	var b [EncodedSize + 2]byte
	b[0] = '"'
	b[1+EncodedSize] = '"'
	_ = id.MarshalTextTo(b[1 : 1+EncodedSize])
	return b[:], nil
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (id *ULID) UnmarshalJSON(data []byte) error {
	if len(data) == EncodedSize+2 {
		if data[0] == '"' {
			// encoding/json does the tokenisation so data[1+EncodedSize]
			// is guaranteed to be '"'
			return parse(data[1:1+EncodedSize], id)
		}
	}
	if len(data) == 4 {
		if string(data) == "null" {
			*id = ULID{}
			return nil
		}
	}
	return ErrInvalidCharacters
}

// Time returns the Unix time in milliseconds encoded in the ULID.
func (id ULID) TimeMs() uint64 {
	return uint64(id[5]) | uint64(id[4])<<8 |
		uint64(id[3])<<16 | uint64(id[2])<<24 |
		uint64(id[1])<<32 | uint64(id[0])<<40
}

// Time returns the Unix time in milliseconds encoded in the ULID,
// and converts it to time.Time.
func (id ULID) Time() time.Time {
	ms := int64(id.TimeMs())
	sec := ms / 1000
	nsec := (ms % 1000) * 1000 * 1000
	return time.Unix(sec, nsec).UTC()
}

// maxTime is the maximum Unix time in milliseconds that can be
// represented in an ULID.
var maxTimeMs = ULID{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}.TimeMs()

// MaxTimeMs returns the maximum Unix time in milliseconds that
// can be encoded in an ULID.
func MaxTimeMs() uint64 { return maxTimeMs }

// now is a convenience function that returns the current
// UTC time in Unix milliseconds. Equivalent to:
//
//	Timestamp(time.Now().UTC())
func NowMs() uint64 { return Timestamp(time.Now().UTC()) }

// Timestamp converts a time.Time to Unix milliseconds.
//
// Because of the way ULID stores time, times from the year
// 10889 produces undefined results.
func Timestamp(t time.Time) uint64 {
	return uint64(t.UnixMilli())
}

// SetTimeMs sets the time component of the ULID to the given Unix time
// in milliseconds.
func (id *ULID) SetTimeMs(ms uint64) error {
	if ms > maxTimeMs {
		return ErrBigTime
	}

	(*id)[0] = byte(ms >> 40)
	(*id)[1] = byte(ms >> 32)
	(*id)[2] = byte(ms >> 24)
	(*id)[3] = byte(ms >> 16)
	(*id)[4] = byte(ms >> 8)
	(*id)[5] = byte(ms)

	return nil
}

// SetTime sets the time component of the ULID
func (id *ULID) SetTime(t time.Time) error {
	return id.SetTimeMs(Timestamp(t.UTC()))
}

// Entropy returns the entropy from the ULID.
func (id ULID) Entropy() []byte {
	e := make([]byte, 10)
	copy(e, id[6:])
	return e
}

// SetEntropy sets the ULID entropy to the passed byte slice.
func (id *ULID) SetEntropy(e []byte) error {
	if len(e) != 10 {
		return fmt.Errorf("ulid: bad data size in SetEntropy")
	}

	copy((*id)[6:], e)
	return nil
}

// AddU64 adds interger to the ULID entropy. allows to build sequential IDs.
// NOTE: this code ignores potential uint64 overflow and wrapping around, which MAY break strict sequentiality of IDs
func (id *ULID) AddU64(x uint64) {
	ptr := id[8:]
	val := binary.BigEndian.Uint64(ptr)
	binary.BigEndian.PutUint64(ptr, val+x)
}

// Compare returns an integer comparing id and other lexicographically.
// The result will be 0 if id==other, -1 if id < other, and +1 if id > other.
func (id ULID) Compare(other ULID) int {
	return bytes.Compare(id[:], other[:])
}

// Scan implements the sql.Scanner interface. It supports scanning
// a string or byte slice.
func (id *ULID) Scan(src any) error {
	switch x := src.(type) {
	case nil:
		*id = Nil
		return nil
	case string:
		return id.UnmarshalText([]byte(x))
	case []byte:
		if len(x) == 16 {
			return id.UnmarshalBinary(x)
		}
		return id.UnmarshalText(x)
	}

	return fmt.Errorf("ulid: cannot convert %T to UUID", src)
}

// Value implements the sql/driver.Valuer interface. This returns the value
// represented as a byte slice. If instead a string is desirable, a wrapper
// type can be created that calls String().
//
// pkg repo has Value() fixed to return string as in uuid pkg.
func (id ULID) Value() (driver.Value, error) {
	if id == (ULID{}) {
		return nil, nil
	}
	return id.String(), nil
}

// Size returns the size of u in bytes and is required by gogoproto
func (id *ULID) Size() int {
	if id == nil {
		return 0
	}
	return 16
}

// Set implements flag.Value interface
func (id *ULID) Set(s string) error {
	uu, err := FromString(s)
	if err != nil {
		return errors.ErrInvalidArg.WithMessagef("malformed ulid %q", s)
	}
	*id = uu
	return nil
}

// Get implements flag.Getter interface
func (id *ULID) Get() any {
	return *id
}

// Type implements pflag.Value interface
func (id *ULID) Type() string {
	return "ulid"
}
