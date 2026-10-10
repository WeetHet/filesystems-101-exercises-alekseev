package logger

import (
	"context"

	"tests-runner/util/logger/handlers"
)

// NewContext returns new context with the given fields attached. If the given context already carries logger fields,
// new ones are merged into existing ones.
func NewContext(ctx context.Context, newFields Fields) context.Context {
	if len(newFields) == 0 {
		return ctx
	}

	fields := make([]handlers.Field, 0, len(newFields))
	for k, v := range newFields {
		fields = append(fields, handlers.MkField(k, v))
	}

	return handlers.NewContext(ctx, fields)
}
