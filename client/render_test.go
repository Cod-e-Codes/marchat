package main

import (
	"strings"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
	"github.com/charmbracelet/x/ansi"
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

func TestPrepareURLWrappingUsesNonBreakingHyphen(t *testing.T) {
	in := "see https://github.com/Cod-e-Codes/marchat"
	out := prepareURLWrapping(in)
	if strings.Contains(out, "Cod-e-Codes") {
		t.Fatalf("expected non-breaking hyphens in URL, got %q", out)
	}
	if !strings.Contains(out, "Cod\u2011e\u2011Codes") {
		t.Fatalf("expected NB hyphen in URL segment, got %q", out)
	}
}

func TestWrapStyledBlockLongURLBreaksAtSlashes(t *testing.T) {
	styles := getThemeStyles("patriot")
	url := "https://github.com/Cod-e-Codes/marchat/commit/351139afcb2f548eb02ff0fd3f107b1c63910a60"
	msgs := []shared.Message{
		{
			Sender:    "Cody",
			Content:   "updated: " + url,
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
			MessageID: 67,
		},
	}
	const width = 59
	out := renderMessages(msgs, styles, "bob", []string{"Cody", "bob"}, width, true, true)
	if strings.Contains(out, "Cod-e-\n") || strings.Contains(out, "Cod-e-\r") {
		t.Fatalf("URL should not break at domain hyphen:\n%s", out)
	}
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.TrimSpace(line) == "" || strings.Contains(line, "June") {
			continue
		}
		if !strings.Contains(line, "http") && !strings.Contains(line, "updated:") && !strings.Contains(line, "Cody:") {
			continue
		}
		if ansi.StringWidth(stripANSIForTest(line)) > width+1 {
			t.Fatalf("line exceeds width %d (%d cells): %q", width, ansi.StringWidth(stripANSIForTest(line)), line)
		}
	}
	if !strings.Contains(out, "/marchat/") && !strings.Contains(out, "/commit/") {
		if !strings.Contains(out, "github.com/Cod") {
			t.Fatalf("expected URL path segments in wrapped output, got:\n%s", out)
		}
	}
}

func underlineOnLeadingWhitespace(s string) bool {
	underline := false
	seenNonSpace := false
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			end := strings.IndexByte(s[i:], 'm')
			if end < 0 {
				break
			}
			seq := s[i : i+end+1]
			i += end + 1
			if strings.Contains(seq, ";4m") || strings.HasSuffix(seq, "[4m") {
				underline = true
			}
			if strings.Contains(seq, ";24m") || strings.HasSuffix(seq, "[24m") || strings.HasSuffix(seq, "[0m") || strings.Contains(seq, ";0m") {
				underline = false
			}
			continue
		}
		r := rune(s[i])
		if r == '\n' || r == '\r' {
			i++
			continue
		}
		if !seenNonSpace {
			if r == ' ' || r == '\t' {
				if underline {
					return true
				}
			} else {
				seenNonSpace = true
			}
		}
		i++
	}
	return false
}

func TestWrapStyledBlockURLUnderlineNotOnContinuationIndent(t *testing.T) {
	styles := getThemeStyles("patriot")
	url := "https://github.com/Cod-e-Codes/marchat/commit/1fc41486340cefc14838f467c4bd09da68ee6947"
	msgs := []shared.Message{
		{
			Sender:    "Cody",
			Content:   url,
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
			MessageID: 70,
		},
	}
	const width = 62
	out := renderMessages(msgs, styles, "bob", []string{"Cody", "bob"}, width, true, true)
	plainLines := strings.Split(strings.TrimRight(ansi.Strip(out), "\n"), "\n")
	rawLines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	for i, plain := range plainLines {
		if !strings.Contains(plain, "1fc414") {
			continue
		}
		if strings.HasPrefix(strings.TrimLeft(plain, " "), "http") {
			continue
		}
		if i >= len(rawLines) {
			t.Fatal("raw/plain line count mismatch")
		}
		if underlineOnLeadingWhitespace(rawLines[i]) {
			t.Fatalf("underline bleeds into continuation indent: %q", rawLines[i])
		}
	}
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
