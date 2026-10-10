package handlers

// This package implements (mostly) zero-allocation logging handlers. A handler is a way
// to format and print log messages.
//
// Structs that describe fields and values are generally copied-and-pasted from log/slog.
//
// The package provides the following handlers:
// 1. stackdriver,
// 2. json,
// 3. text,
// 4. freeform text,
// 5. junit,
// 6. fanout.
//
// The output produced by the JSON handler matches that of slog's JSON handler.
//
// The output of text and freeform text formatters matches that of logrus' text formatter
// configured to write to non-TTY output files and to TTYs, respectively.
//
// The output of the JUnit handler can be piped to gotestsum to produce junit.xml files.
