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

func TestIsTranscriptSystemNotice(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"This command requires admin privileges", false},
		{"Unknown command: :foo", false},
		{"Usage: :kick <username>", false},
		{"Search results for 'hi' (1 found):", true},
		{searchNoResultsPrefix + " test", true},
		{"Joined channel #dev", true},
		{"Pinned messages (2):\n  #1", true},
		{"Available themes:\n\n  • system", true},
		{e2eSearchNoResultsHint, true},
	}
	for _, tt := range tests {
		if got := isTranscriptSystemNotice(tt.content); got != tt.want {
			t.Fatalf("%q: got %v want %v", tt.content, got, tt.want)
		}
	}
}

func TestSystemBannerTextAddsErrorPrefix(t *testing.T) {
	got := systemBannerText("Edit failed: denied")
	if !strings.HasPrefix(got, "[ERROR]") {
		t.Fatalf("got %q", got)
	}
	if systemBannerText("Theme changed") != "Theme changed" {
		t.Fatal("info line should pass through")
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

func TestMessageLessPersistedBeforeClientSystem(t *testing.T) {
	now := time.Now()
	a := shared.Message{Sender: "System", Content: "usage", CreatedAt: now.Add(time.Second), MessageID: -1}
	b := shared.Message{Sender: "alice", Content: "hello", CreatedAt: now, MessageID: 42}
	if messageLess(a, b) {
		t.Fatal("client system line must not sort before persisted chat")
	}
	if !messageLess(b, a) {
		t.Fatal("persisted chat must sort before client system line")
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
	if !strings.Contains(ansi.Strip(out), "/marchat/") && !strings.Contains(ansi.Strip(out), "/commit/") {
		if !strings.Contains(ansi.Strip(out), "github.com/Cod") {
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

func hasHyperlinkANSI(s string) bool {
	return strings.Contains(s, "[4m") || strings.Contains(s, ";4m") || strings.Contains(s, "[4;")
}

func TestMarkURLsForWrapInsertsSentinels(t *testing.T) {
	in := "see https://example.com/path ok"
	out := markURLsForWrap(in)
	if !strings.Contains(out, string(urlStartMarker)+"https://example.com/path"+string(urlEndMarker)) {
		t.Fatalf("expected URL sentinels, got %q", out)
	}
}

func TestApplyURLMarkersAcrossWrappedLines(t *testing.T) {
	styles := getThemeStyles("patriot")
	url := "https://github.com/Cod-e-Codes/marchat/commit/8b765b04f82a16c51128261c2fef88c6fef05a61"
	marked := markURLsForWrap(prepareURLWrapping(url))
	wrapped := ansi.Wrap(marked, 30, wrapBreakpoints)
	open := false
	var styled strings.Builder
	for _, line := range strings.Split(wrapped, "\n") {
		styled.WriteString(applyURLMarkers(line, styles, &open))
	}
	if open {
		t.Fatal("expected URL span to close after processing all wrapped lines")
	}
	plain := ansi.Strip(styled.String())
	if strings.ContainsRune(plain, urlStartMarker) || strings.ContainsRune(plain, urlEndMarker) {
		t.Fatalf("URL sentinels leaked into styled output: %q", plain)
	}
	if plain != prepareURLWrapping(url) {
		t.Fatalf("styled plain text mismatch:\n got %q\nwant %q", plain, prepareURLWrapping(url))
	}
	open = false
	for _, line := range strings.Split(wrapped, "\n") {
		if ansi.Strip(line) == "" {
			continue
		}
		segment := applyURLMarkers(line, styles, &open)
		if !hasHyperlinkANSI(segment) {
			t.Fatalf("expected hyperlink style on URL segment %q", line)
		}
	}
}

func TestWrapStyledBlockWrappedURLSegmentsStyled(t *testing.T) {
	styles := getThemeStyles("patriot")
	url := "https://github.com/Cod-e-Codes/marchat/commit/8b765b04f82a16c51128261c2fef88c6fef05a61"
	msgs := []shared.Message{
		{
			Sender:    "Cody",
			Content:   url,
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
			MessageID: 73,
		},
	}
	const width = 62
	out := renderMessages(msgs, styles, "bob", []string{"Cody", "bob"}, width, true, true)
	plain := ansi.Strip(out)
	if strings.ContainsRune(plain, urlStartMarker) || strings.ContainsRune(plain, urlEndMarker) {
		t.Fatalf("URL sentinels leaked into transcript: %q", plain)
	}
	if !strings.Contains(plain, "8b765b04f82a16c51128261c2fef88c6fef05a61") {
		t.Fatalf("full URL missing from output: %q", plain)
	}
	rawLines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(ansi.Strip(line))
		if trimmed == "" || strings.Contains(trimmed, "June") || strings.Contains(trimmed, "Cody:") {
			continue
		}
		if strings.Contains(trimmed, "github.com") || strings.Contains(trimmed, "marchat") || strings.Contains(trimmed, "8b765b") {
			if !hasHyperlinkANSI(line) {
				t.Fatalf("wrapped URL segment missing hyperlink style: %q", line)
			}
		}
	}
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
	if !strings.Contains(ansi.Strip(out), "alice: hi") {
		t.Fatalf("short message missing: %q", out)
	}
	msgLines := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(ansi.Strip(line), "alice:") {
			msgLines++
		}
	}
	if msgLines != 1 {
		t.Fatalf("short message should be a single chat line, got %d: %q", msgLines, out)
	}
}
