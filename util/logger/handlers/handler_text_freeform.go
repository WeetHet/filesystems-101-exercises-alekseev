package handlers

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
	"unsafe"
)

func NewFreeformTextHandler(out io.Writer) Handler {
	return &freeformTextHandler{
		out:       out,
		startedAt: time.Now(),
	}
}

type freeformTextHandler struct {
	lock sync.Mutex
	out  io.Writer

	startedAt time.Time
	fields    []Field
}

func (h *freeformTextHandler) Handle(ctx context.Context, _level Level, msg string) {
	msg = truncateMsg(msg)

	b := h.getTextBuilder()
	defer h.putTextBuilder(&b)

	level := _level.String()
	if len(level) > 4 {
		level = level[:4]
	}
	fmt.Fprintf(&b, "%s[%04d] %-45s",
		level,
		int(time.Since(h.startedAt).Seconds()),
		msg)

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

func (h *freeformTextHandler) WithFields(fields []Field) Handler {
	return &freeformTextHandler{
		out:       h.out,
		startedAt: h.startedAt,
		fields:    joinFields(h.fields, fields),
	}
}

func (h *freeformTextHandler) getTextBuilder() textBuilder {
	buf := bufferPool.Get().(*byte)
	return textBuilder{
		buf: unsafe.Slice(buf, pooledBufferSize)[:0],
	}
}

func (h *freeformTextHandler) putTextBuilder(b *textBuilder) {
	if cap(b.buf) == pooledBufferSize {
		bufferPool.Put(unsafe.SliceData(b.buf))
	}
}
