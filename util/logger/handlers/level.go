package handlers

import (
	"fmt"
	"strconv"
	"strings"
)

type Level int

const (
	LevelDebug Level = -4
	LevelInfo  Level = 0
	LevelWarn  Level = 4
	LevelError Level = 8
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return fmt.Sprintf("logger.Level(%d)", l)
	}
}

func (l Level) MarshalJSON() ([]byte, error) {
	// AppendQuote is sufficient for JSON-encoding all Level strings.
	// They don't contain any runes that would produce invalid JSON
	// when escaped.
	return strconv.AppendQuote(nil, l.String()), nil
}

func (l *Level) UnmarshalJSON(data []byte) error {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	return l.parse(s)
}

func (l Level) AppendText(b []byte) ([]byte, error) {
	return append(b, l.String()...), nil
}

func (l Level) MarshalText() ([]byte, error) {
	return l.AppendText(nil)
}

func (l *Level) UnmarshalText(data []byte) error {
	return l.parse(string(data))
}

func (l *Level) parse(s string) (err error) {
	switch strings.ToUpper(s) {
	case "DEBUG":
		*l = LevelDebug
	case "INFO":
		*l = LevelInfo
	case "WARN", "WARNING":
		*l = LevelWarn
	case "ERROR":
		*l = LevelError
	default:
		return fmt.Errorf("unknown logger.Level %q", s)
	}
	return nil
}

func (l *Level) Set(x string) error {
	return l.UnmarshalText([]byte(x))
}

func (l Level) Type() string {
	return "logger.Level"
}
