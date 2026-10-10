package handlers

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	"unsafe"

	"tests-runner/util/util/ptr"
)

func NewTextHandler(out io.Writer) Handler {
	return &textHandler{
		out:       out,
		startedAt: time.Now(),
	}
}

type textHandler struct {
	lock sync.Mutex
	out  io.Writer

	startedAt time.Time
	fields    []Field
}

func (h *textHandler) Handle(ctx context.Context, level Level, msg string) {
	msg = truncateMsg(msg)

	b := h.getTextBuilder()
	defer h.putTextBuilder(&b)

	b.appendField(MkField("time", time.Now()))
	b.appendByte(' ')
	b.appendField(MkField("level", strings.ToLower(level.String())))
	b.appendByte(' ')
	b.appendField(MkField("msg", msg))

	for _, a := range h.fields {
		b.appendByte(' ')
		b.appendField(a)
	}
	for a := range fieldsFromContext(ctx) {
		b.appendByte(' ')
		b.appendField(a)
	}

	b.appendByte('\n')

	h.lock.Lock()
	_, _ = h.out.Write(b.buf)
	h.lock.Unlock()
}

func (h *textHandler) WithFields(fields []Field) Handler {
	return &textHandler{
		out:       h.out,
		startedAt: h.startedAt,
		fields:    joinFields(h.fields, fields),
	}
}

func joinFields(existing, new []Field) (updated []Field) {
	if len(existing) == 0 {
		updated = new
	} else {
		m := make(map[string]Field)
		for _, a := range existing {
			m[a.key] = a
		}
		for _, a := range new {
			m[a.key] = a
		}

		updated = make([]Field, 0, len(m))
		for _, a := range m {
			updated = append(updated, a)
		}
	}
	slices.SortFunc(updated, func(a, b Field) int {
		return strings.Compare(a.key, b.key)
	})
	return updated
}

func truncateMsg(msg string) string {
	const maxLen = 32 * 1024
	if len(msg) > maxLen {
		msg = fmt.Sprintf("%s...(truncated)", msg[:maxLen])
	}
	return msg
}

func (h *textHandler) getTextBuilder() textBuilder {
	buf := bufferPool.Get().(*byte)
	return textBuilder{
		buf: unsafe.Slice(buf, pooledBufferSize)[:0],
	}
}

func (h *textHandler) putTextBuilder(b *textBuilder) {
	if cap(b.buf) == pooledBufferSize {
		bufferPool.Put(unsafe.SliceData(b.buf))
	}
}

const pooledBufferSize = 2048

var bufferPool = sync.Pool{
	New: func() any {
		buf := ptr.Malloc(pooledBufferSize)
		return &buf[0]
	},
}

type textBuilder struct {
	buf []byte
}

func (b *textBuilder) appendField(a Field) {
	k, v := a.key, a.value

	b.appendString(k)
	b.appendByte('=')

	switch v.Kind() {
	case KindString:
		b.appendString(v.String())

	case KindInt64:
		b.appendInt64(v.Int64())
	case KindUint64:
		b.appendUint64(v.Uint64())
	case KindFloat64:
		b.appendFloat64(v.Float64())

	case KindBool:
		b.appendBool(v.Bool())

	case KindTime:
		b.appendTime(v.Time())
	case KindDuration:
		b.appendDuration(v.Duration())

	case KindAny:
		switch v := v.Any().(type) {
		case error:
			b.appendString(v.Error())
		case fmt.Stringer: // for mostly free-form text, prefer String() to JSON
			b.appendString(v.String())
		case []byte:
			b.appendBytes(v)

		default:
			// Not nice for a "text" formatter, but whatever...
			b.appendString(fmt.Sprintf("%+v", v))
		}

	default:
		panic(fmt.Sprintf("unsupported field kind %q", v.Kind()))
	}
}

func (b *textBuilder) appendBool(x bool) {
	if x {
		b.appendString("true")
	} else {
		b.appendString("false")
	}
}

func (b *textBuilder) appendInt64(x int64) {
	b.buf = strconv.AppendInt(b.buf, x, 10)
}

func (b *textBuilder) appendUint64(x uint64) {
	b.buf = strconv.AppendUint(b.buf, x, 10)
}

func (b *textBuilder) appendFloat64(x float64) {
	b.buf = strconv.AppendFloat(b.buf, x, 'f', -1, 64)
}

func (b *textBuilder) appendTime(t time.Time) {
	t = t.UTC()
	// NOTE: time.AppendFormat() truncates trailing zeros, thus
	// time.RFC3339Nano can implement "time.RFC3339Milli".
	t = t.Truncate(time.Millisecond)

	b.buf = t.AppendFormat(b.buf, time.RFC3339Nano)
}

func (b *textBuilder) appendDuration(x time.Duration) {
	b.appendString(x.String())
}

func (b *textBuilder) appendByte(c byte) {
	b.buf = append(b.buf, c)
}

func (b *textBuilder) appendBytes(s []byte) {
	b.appendString(string(s))
}

func (b *textBuilder) Write(p []byte) (n int, err error) {
	b.appendBytes(p)
	return len(p), nil
}

func (b *textBuilder) appendString(s string) {
	for len(s) > 0 {
		r, width := utf8.DecodeRuneInString(s)
		if width == 1 && r == utf8.RuneError {
			b.buf = append(b.buf, `\x`...)
			b.buf = append(b.buf, lowerhex[s[0]>>4])
			b.buf = append(b.buf, lowerhex[s[0]&0xF])
		} else {
			b.buf = appendEscapedRune(b.buf, r)
		}
		s = s[width:]
	}
}

const lowerhex = "0123456789abcdef"

func appendEscapedRune(buf []byte, r rune) []byte {
	if r == '\\' {
		buf = append(buf, '\\')
		buf = append(buf, byte(r))
		return buf
	}
	if strconv.IsPrint(r) {
		return utf8.AppendRune(buf, r)
	}
	switch r {
	case '\a':
		buf = append(buf, `\a`...)
	case '\b':
		buf = append(buf, `\b`...)
	case '\f':
		buf = append(buf, `\f`...)
	case '\n':
		buf = append(buf, `\n`...)
	case '\r':
		buf = append(buf, `\r`...)
	case '\t':
		buf = append(buf, `\t`...)
	case '\v':
		buf = append(buf, `\v`...)
	default:
		switch {
		case r < ' ' || r == 0x7f:
			buf = append(buf, `\x`...)
			buf = append(buf, lowerhex[byte(r)>>4])
			buf = append(buf, lowerhex[byte(r)&0xF])
		case !utf8.ValidRune(r):
			r = 0xFFFD
			fallthrough
		case r < 0x10000:
			buf = append(buf, `\u`...)
			for s := 12; s >= 0; s -= 4 {
				buf = append(buf, lowerhex[r>>uint(s)&0xF])
			}
		default:
			buf = append(buf, `\U`...)
			for s := 28; s >= 0; s -= 4 {
				buf = append(buf, lowerhex[r>>uint(s)&0xF])
			}
		}
	}
	return buf
}
