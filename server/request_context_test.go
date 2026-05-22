package server

import (
	"net/http"
	"testing"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name    string
		request *http.Request
		trusted string
		want    string
	}{
		{
			name: "ignores X-Forwarded-For without trusted proxy",
			request: &http.Request{
				Header:     http.Header{"X-Forwarded-For": []string{"192.168.1.1"}},
				RemoteAddr: "203.0.113.9:12345",
			},
			want: "203.0.113.9",
		},
		{
			name: "honors X-Forwarded-For from trusted proxy",
			request: &http.Request{
				Header:     http.Header{"X-Forwarded-For": []string{"192.168.1.1"}},
				RemoteAddr: "10.0.0.1:12345",
			},
			trusted: "10.0.0.1",
			want:    "192.168.1.1",
		},
		{
			name: "X-Forwarded-For multiple IPs via trusted proxy",
			request: &http.Request{
				Header:     http.Header{"X-Forwarded-For": []string{"192.168.1.1, 10.0.0.1, 172.16.0.1"}},
				RemoteAddr: "10.0.0.1:12345",
			},
			trusted: "10.0.0.1",
			want:    "192.168.1.1",
		},
		{
			name: "X-Real-IP via trusted proxy",
			request: &http.Request{
				Header:     http.Header{"X-Real-Ip": []string{"203.0.113.1"}},
				RemoteAddr: "10.0.0.1:12345",
			},
			trusted: "10.0.0.1",
			want:    "203.0.113.1",
		},
		{
			name: "RemoteAddr fallback",
			request: &http.Request{
				RemoteAddr: "192.168.1.100:12345",
			},
			want: "192.168.1.100",
		},
		{
			name: "RemoteAddr without port",
			request: &http.Request{
				RemoteAddr: "192.168.1.100",
			},
			want: "192.168.1.100",
		},
		{
			name:    "No IP information",
			request: &http.Request{},
			want:    "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetRequestContextForTests()
			if tt.trusted != "" {
				t.Setenv("MARCHAT_TRUSTED_PROXIES", tt.trusted)
			} else {
				t.Setenv("MARCHAT_TRUSTED_PROXIES", "")
			}
			got := getClientIP(tt.request)
			if got != tt.want {
				t.Errorf("getClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckWebSocketOrigin(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		host    string
		allowed string
		want    bool
	}{
		{name: "empty origin", want: true},
		{
			name:   "exact same host",
			origin: "https://chat.example.com",
			host:   "chat.example.com",
			want:   true,
		},
		{
			name:   "substring host bypass blocked",
			origin: "https://evil-chat.example.com.attacker.net",
			host:   "chat.example.com",
			want:   false,
		},
		{
			name:   "substring localhost bypass blocked",
			origin: "https://localhost.evil.example",
			host:   "localhost:8080",
			want:   false,
		},
		{
			name:   "loopback alias",
			origin: "http://127.0.0.1:8080",
			host:   "localhost:8080",
			want:   true,
		},
		{
			name:    "explicit allowlist",
			origin:  "https://app.other.example",
			host:    "chat.example.com",
			allowed: "https://app.other.example",
			want:    true,
		},
		{
			name:   "invalid origin URL",
			origin: "not-a-url",
			host:   "chat.example.com",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetRequestContextForTests()
			if tt.allowed != "" {
				t.Setenv("MARCHAT_ALLOWED_ORIGINS", tt.allowed)
			} else {
				t.Setenv("MARCHAT_ALLOWED_ORIGINS", "")
			}
			r := &http.Request{Host: tt.host, Header: make(http.Header)}
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			got := checkWebSocketOrigin(r)
			if got != tt.want {
				t.Errorf("checkWebSocketOrigin() = %v, want %v (origin=%q host=%q)", got, tt.want, tt.origin, tt.host)
			}
		})
	}
}
