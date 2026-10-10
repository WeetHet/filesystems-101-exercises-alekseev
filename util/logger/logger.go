package logger

import (
	"context"
	"fmt"
	"os"

	"tests-runner/util/logger/handlers"
)

// NewNoop returns a new instance of the Logger that doesn't output any
// messages (should be used as default in packages when no logger is provided).
func NewNoop() Logger {
	return &loggerImpl{h: handlers.Nop}
}

// New returns a new instance of the Logger that is backed by a log/slog with
// default settings. Log level is set to Info.
func New() Logger {
	return &loggerImpl{
		h:   handlers.NewTextHandler(os.Stderr),
		lvl: Info,
	}
}

func NewFromHandler(h handlers.Handler) Logger {
	return &loggerImpl{
		h:   h,
		lvl: Info,
	}
}

// This logger works almost like slog.Logger, but it skips runtime.Callers()
// and does not support attribute groups.
type loggerImpl struct {
	h   handlers.Handler
	lvl Level
}

// Logf logs the given message with specified log level.
func (l *loggerImpl) Logf(ctx context.Context, level Level, format string, args ...any) {
	if level >= l.lvl {
		l.handle(ctx, level, format, args...)
	}
}

// Debugf logs the given message with Debug log level.
// Same as Logf(ctx, Debug, format, args...), but we copy-paste the code intentionally to help inlining
func (l *loggerImpl) Debugf(ctx context.Context, format string, args ...any) {
	if Debug >= l.lvl {
		l.handle(ctx, Debug, format, args...)
	}
}

// Infof logs the given message with Info log level.
// Same as Logf(ctx, Info, format, args...), but we copy-paste the code intentionally to help inlining
func (l *loggerImpl) Infof(ctx context.Context, format string, args ...any) {
	if Info >= l.lvl {
		l.handle(ctx, Info, format, args...)
	}
}

// Warningf logs the given message with Warning log level.
// Same as Logf(ctx, Warning, format, args...), but we copy-paste the code intentionally to help inlining
func (l *loggerImpl) Warningf(ctx context.Context, format string, args ...any) {
	if Warning >= l.lvl {
		l.handle(ctx, Warning, format, args...)
	}
}

// Warnf logs the given message with Warning log level.
// Same as Logf(ctx, Warning, format, args...), but we copy-paste the code intentionally to help inlining
func (l *loggerImpl) Warnf(ctx context.Context, format string, args ...any) {
	if Warning >= l.lvl {
		l.handle(ctx, Warning, format, args...)
	}
}

// Errorf logs the given message with Error log level.
// Same as Logf(ctx, Error, format, args...), but we copy-paste the code intentionally to help inlining
func (l *loggerImpl) Errorf(ctx context.Context, format string, args ...any) {
	if Error >= l.lvl {
		l.handle(ctx, Error, format, args...)
	}
}

func (l *loggerImpl) handle(ctx context.Context, level Level, format string, args ...any) {
	var msg string
	if len(args) == 0 {
		msg = format
	} else {
		msg = fmt.Sprintf(format, args...)
	}
	l.h.Handle(ctx, level, msg)
}

// WithError returns an instance of the logger with error key set to err.
func (l *loggerImpl) WithError(err error) Logger {
	if err != nil {
		return l.WithField(KeyError, err.Error())
	} else {
		return l.WithField(KeyError, "<nil>")
	}
}

// WithField returns an instance of the logger with field set to val.
func (l *loggerImpl) WithField(field string, val any) Logger {
	return &loggerImpl{
		h:   l.h.WithFields([]handlers.Field{handlers.MkField(field, val)}),
		lvl: l.lvl,
	}
}

// WithFields returns an instance of the logger with given key/value pairs
// added to log context.
func (l *loggerImpl) WithFields(newFields Fields) Logger {
	fields := make([]handlers.Field, 0, len(newFields))
	for field, val := range newFields {
		fields = append(fields, handlers.MkField(field, val))
	}
	return &loggerImpl{
		h:   l.h.WithFields(fields),
		lvl: l.lvl,
	}
}

func (l *loggerImpl) WithContextFields(ctx context.Context) Logger {
	return &loggerImpl{
		h:   handlers.WithContextFields(ctx, l.h),
		lvl: l.lvl,
	}
}

// WithLevel returns an instance of the logger with the log level set to level.
func (l *loggerImpl) WithLevel(level Level) Logger {
	return &loggerImpl{h: l.h, lvl: level}
}

func (l *loggerImpl) Enabled(level Level) bool {
	return level >= l.lvl
}
