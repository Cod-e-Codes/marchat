package server

import "strings"

// contentContainsNUL reports whether s contains a NUL byte. Postgres rejects NUL in TEXT;
// SQLite accepts it. The server rejects NUL before persist for cross-dialect consistency.
func contentContainsNUL(s string) bool {
	return strings.Contains(s, "\x00")
}
