package handlers

import (
	"context"
)

type Handler interface {
	Handle(ctx context.Context, level Level, msg string)
	WithFields(fields []Field) Handler
}
