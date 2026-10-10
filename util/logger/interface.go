package logger

import (
	"tests-runner/util/logger/handlers"
)

// Logger is an interface that describes an abstract structured logger without
// enforcing any particular implementation.
// It should accepted by type consturctors or package functions and
// instantiated with a specific implementation by a service initialization
// code.
// NOTE: while we have single logger implementation (logrus) - we set Logger to *logger, rather than interface.
// Interface costs quite a lot: compiler can't inline + perform escape analysis and ALWAYS allocates slice with args on heap.
// If switching back to interface, than Logrus should be made private
type Logger = *loggerImpl

// Level defines a log level.
type Level = handlers.Level

const (
	Error   = handlers.LevelError
	Warning = handlers.LevelWarn
	Info    = handlers.LevelInfo
	Debug   = handlers.LevelDebug
)

// Fields contains a collection of key/value pairs that gets added to log messages.
type Fields = map[string]any
