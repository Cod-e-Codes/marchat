package server

import "testing"

func TestPlaintextContentEmpty(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		encrypted bool
		want      bool
	}{
		{name: "empty", content: "", encrypted: false, want: true},
		{name: "whitespace", content: " \t\n ", encrypted: false, want: true},
		{name: "non-empty", content: "hello", encrypted: false, want: false},
		{name: "encrypted empty opaque", content: "", encrypted: true, want: false},
		{name: "encrypted whitespace opaque", content: "   ", encrypted: true, want: false},
		{name: "encrypted non-empty blob", content: "Y2lwaGVydGV4dA==", encrypted: true, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := plaintextContentEmpty(tt.content, tt.encrypted); got != tt.want {
				t.Fatalf("plaintextContentEmpty(%q, %v) = %v, want %v", tt.content, tt.encrypted, got, tt.want)
			}
		})
	}
}
