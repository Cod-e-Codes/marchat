package main

import "testing"

func TestSanitizeServerURL(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"wss://h/ws", "wss://h/ws"},
		{`'wss://h/ws'`, "wss://h/ws"},
		{`"wss://h/ws"`, "wss://h/ws"},
		{"  wss://h/ws  ", "wss://h/ws"},
		{"\u2018wss://h/ws\u2019", "wss://h/ws"},
	}
	for _, tt := range tests {
		if got := sanitizeServerURL(tt.in); got != tt.want {
			t.Errorf("sanitizeServerURL(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
