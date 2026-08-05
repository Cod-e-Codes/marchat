package server

import "strings"

// contentContainsNUL reports whether s contains a NUL byte. Postgres rejects NUL in TEXT;
// SQLite accepts it. The server rejects NUL before persist for cross-dialect consistency.
func contentContainsNUL(s string) bool {
	return strings.Contains(s, "\x00")
}

// plaintextContentEmpty reports whether unencrypted text/dm/edit content is empty or
// whitespace-only. Encrypted payloads are opaque and are never treated as empty here.
func plaintextContentEmpty(content string, encrypted bool) bool {
	if encrypted {
		return false
	}
	return strings.TrimSpace(content) == ""
}
