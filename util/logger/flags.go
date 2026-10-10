package logger

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"tests-runner/util/logger/handlers"
)

var levels = []string{
	Error.String(),
	Warning.String(),
	Info.String(),
	Debug.String(),
}

// Formatter describes which formatter to use to output messages
type Formatter int

const (
	// FormatterFreeformText formats log lines to mostly aligned text like
	// WARN[0600] hello                      k=v
	FormatterFreeformText Formatter = iota
	// FormatterText is a simple plain-text formatter that uses unquoted k=v format:
	// time=2006-01-02T15:04:05.000Z level=info msg=hello k=v
	FormatterText
	// FormatterJSON is a formatter that output log messages as JSON objects
	FormatterJSON
	// FormatterStackdriver is a Stackdriver Cloud Logging compatible formatter
	FormatterStackdriver
)

// String returns string representation of the formatter
func (f Formatter) String() string {
	switch f {
	case FormatterFreeformText:
		return "freeform-text"
	case FormatterText:
		return "text"
	case FormatterJSON:
		return "json"
	case FormatterStackdriver:
		return "stackdriver"
	default:
		return "<unknown>"
	}
}

// Set implements flag.Value interface
func (f *Formatter) Set(s string) error {
	switch strings.ToLower(s) {
	//case "freeform-text":
	// don't expose "freeform-text" this to the CLI
	case "text":
		*f = FormatterText
	default:
		return fmt.Errorf("unknown formatter %q", s)
	}
	return nil
}

// UnmarshalText implements encoding.TextUnmarshaler interface
func (f *Formatter) UnmarshalText(b []byte) error {
	return f.Set(string(b))
}

// Type implements pflag.Value interface
func (f Formatter) Type() string {
	return "formatter"
}

var formatters = []string{
	// don't expose "freeform-text" this to the CLI
	FormatterText.String(),
}

// Flags holds all logging related flags
type Flags struct {
	Level     Level     `arg:"env:LOG_LEVEL"`
	Formatter Formatter `arg:"env:LOG_FORMATTER"`
}

// InjectFlags injects logger related flags into the given flag set.
func InjectFlags(fs *pflag.FlagSet, f *Flags) {
	f.Level = Info
	f.Formatter = FormatterFreeformText

	fs.VarP(&f.Level, "log.level", "", "One of the following log levels: "+strings.Join(levels, ", "))
	fs.VarP(&f.Formatter, "log.formatter", "", "One of the following formatters: "+strings.Join(formatters, ", "))
}

// NewFromFlags returns new instance of the logger initialized from the
// previously parsed command line flags.
func NewFromFlags(f *Flags) Logger {
	var h handlers.Handler
	switch f.Formatter {
	case FormatterFreeformText:
		h = handlers.NewFreeformTextHandler(os.Stderr)
	case FormatterText:
		h = handlers.NewTextHandler(os.Stderr)
	}

	return &loggerImpl{
		h:   h,
		lvl: f.Level,
	}
}
