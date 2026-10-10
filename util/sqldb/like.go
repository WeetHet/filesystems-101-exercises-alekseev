package sqldb

import "strings"

// EscapeLikeQuery escapes '%', '_' and '[' (SQL Server) to make sure
// user can not affect the pattern and we really search for symbols entered.
// NOTE: WHERE should contain ESCAPE '\' clause as MS SQL have no default escape mode.
func EscapeLikeQuery(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`) // escape escape-char first
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	s = strings.ReplaceAll(s, `[`, `\[`)
	return s
}
