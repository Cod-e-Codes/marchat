package main

import (
	"strings"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
)

func TestSystemLineSeverityClass(t *testing.T) {
	tests := []struct {
		content string
		want    systemLineSeverity
	}{
		{"Plugin ok", systemLineInfo},
		{"Unknown plugin subcommand", systemLineErr},
		{"[ERROR] x", systemLineErr},
		{"[WARN] x", systemLineWarn},
		{"invalid input", systemLineErr},
		{"Operation failed", systemLineErr},
		{"No failure here", systemLineInfo},
	}
	for _, tt := range tests {
		if got := systemLineSeverityClass(tt.content); got != tt.want {
			t.Fatalf("%q: got %v want %v", tt.content, got, tt.want)
		}
	}
}

func TestRenderMessagesSystemUsesSemanticStyle(t *testing.T) {
	now := time.Now()
	msgs := []shared.Message{
		{Sender: "System", Content: "All quiet", CreatedAt: now, Type: shared.TextMessage},
		{Sender: "System", Content: "Unknown command", CreatedAt: now.Add(time.Second), Type: shared.TextMessage},
	}
	styles := getThemeStyles("patriot")
	out := renderMessages(msgs, styles, "u", []string{"u"}, 80, true, false)
	if !strings.Contains(out, "All quiet") || !strings.Contains(out, "Unknown command") {
		t.Fatalf("expected both lines in output: %q", out)
	}
}
