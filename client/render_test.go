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

func TestResolveReactionEmojiThumbsAliases(t *testing.T) {
	if got := resolveReactionEmoji("thumbsup"); got != "👍" {
		t.Fatalf("thumbsup: got %q want thumbs up", got)
	}
	if got := resolveReactionEmoji("THUMBSDOWN"); got != "👎" {
		t.Fatalf("THUMBSDOWN: got %q want thumbs down", got)
	}
}

func TestWrapStyledBlockLongMessage(t *testing.T) {
	styles := getThemeStyles("patriot")
	long := strings.Repeat("word ", 40)
	msgs := []shared.Message{
		{
			Sender:    "alice",
			Content:   long,
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
			MessageID: 1,
		},
	}
	const width = 40
	out := renderMessages(msgs, styles, "bob", []string{"alice", "bob"}, width, true, false)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected wrapped output across multiple lines, got %d lines: %q", len(lines), out)
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Allow date header lines to exceed width; message body lines should fit.
		if strings.Contains(line, "word") && len([]rune(stripANSIForTest(line))) > width+2 {
			t.Fatalf("line exceeds viewport width %d: %q", width, line)
		}
	}
}

func stripANSIForTest(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		if r == '\x1b' {
			esc = true
			continue
		}
		if esc {
			if r == 'm' {
				esc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func TestWrapStyledBlockShortMessageUnchanged(t *testing.T) {
	styles := getThemeStyles("patriot")
	msgs := []shared.Message{
		{Sender: "alice", Content: "hi", CreatedAt: time.Now(), Type: shared.TextMessage},
	}
	out := renderMessages(msgs, styles, "bob", []string{"alice", "bob"}, 80, true, false)
	if !strings.Contains(out, "alice: hi") {
		t.Fatalf("short message missing: %q", out)
	}
	msgLines := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "alice:") {
			msgLines++
		}
	}
	if msgLines != 1 {
		t.Fatalf("short message should be a single chat line, got %d: %q", msgLines, out)
	}
}
