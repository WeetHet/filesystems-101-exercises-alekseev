package handlers

import (
	"context"
	"iter"
	"slices"
	"strings"
)

type contextKey struct{}

type loggerCtx struct {
	fields []Field
}

// NewContext returns a new context with the given fields added.
// New fields take precedence over fields in ctx.
func NewContext(ctx context.Context, newFields []Field) context.Context {
	fieldsMap := make(map[string]Field)

	old, _ := ctx.Value(contextKey{}).(*loggerCtx)
	if old != nil {
		for _, a := range old.fields {
			fieldsMap[a.key] = a
		}
	}
	for _, a := range newFields {
		fieldsMap[a.key] = a
	}

	new := &loggerCtx{fields: make([]Field, 0, len(fieldsMap))}
	for _, a := range fieldsMap {
		new.fields = append(new.fields, a)
	}
	slices.SortFunc(new.fields, func(a, b Field) int {
		return strings.Compare(a.key, b.key)
	})

	return context.WithValue(ctx, contextKey{}, new)
}

func WithContextFields(ctx context.Context, h Handler) Handler {
	lctx, _ := ctx.Value(contextKey{}).(*loggerCtx)
	if lctx == nil {
		return h
	} else {
		return h.WithFields(lctx.fields)
	}
}

func fieldsFromContext(ctx context.Context) iter.Seq[Field] {
	return func(yield func(f Field) bool) {
		ctxFields, _ := ctx.Value(contextKey{}).(*loggerCtx)
		if ctxFields == nil {
			return
		}

		for _, a := range ctxFields.fields {
			if !yield(a) {
				return
			}
		}
	}
}
