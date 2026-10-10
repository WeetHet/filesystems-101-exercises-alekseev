package handlers

import (
	"context"
)

var Nop Handler = &nopHandler{}

type nopHandler struct{}

func (h *nopHandler) Handle(ctx context.Context, level Level, msg string) {
}

func (h *nopHandler) WithFields(fields []Field) Handler {
	return h
}
