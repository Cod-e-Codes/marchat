package main

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/viewport"
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
	// Strip CSI and OSC 8 hyperlink sequences so width checks match terminal cells.
	s = ansi.Strip(s)
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == ']' {
			end := strings.IndexByte(s[i:], '\a')
			if end < 0 {
				b.WriteByte(s[i])
				i++
				continue
			}
			i += end + 1
			continue
		}
		b.WriteByte(s[i])
		i++
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
		tolerance := 1
		if strings.Contains(line, "[id:") {
			tolerance = 4 // metadata suffix on wrapped URL continuation lines
		}
		if ansi.StringWidth(stripANSIForTest(line)) > width+tolerance {
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

func hasOSC8Hyperlink(s string) bool {
	return strings.Contains(s, "\x1b]8;;")
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
	state := urlMarkerState{urlQueue: parseMarkedURLs(marked)}
	var styled strings.Builder
	for _, line := range strings.Split(wrapped, "\n") {
		styled.WriteString(applyURLMarkers(line, styles, &state))
	}
	if state.open {
		t.Fatal("expected URL span to close after processing all wrapped lines")
	}
	plain := ansi.Strip(styled.String())
	if strings.ContainsRune(plain, urlStartMarker) || strings.ContainsRune(plain, urlEndMarker) {
		t.Fatalf("URL sentinels leaked into styled output: %q", plain)
	}
	if plain != prepareURLWrapping(url) {
		t.Fatalf("styled plain text mismatch:\n got %q\nwant %q", plain, prepareURLWrapping(url))
	}
	state = urlMarkerState{urlQueue: parseMarkedURLs(marked)}
	for _, line := range strings.Split(wrapped, "\n") {
		if ansi.Strip(line) == "" {
			continue
		}
		segment := applyURLMarkers(line, styles, &state)
		if !hasOSC8Hyperlink(segment) {
			t.Fatalf("expected OSC 8 hyperlink on URL segment %q", line)
		}
		if !hasHyperlinkANSI(segment) {
			t.Fatalf("expected hyperlink underline on URL segment %q", line)
		}
		if strings.Contains(segment, "Cod\u2011e") || strings.Contains(segment, "Cod%E2%80%91") {
			t.Fatalf("OSC 8 href must use ASCII hyphens, not non-breaking hyphens: %q", segment)
		}
		if !strings.Contains(segment, "\x1b]8;;https://github.com/Cod-e-Codes/") {
			t.Fatalf("expected ASCII hyphen in OSC 8 href, got %q", segment)
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
			if !hasOSC8Hyperlink(line) {
				t.Fatalf("wrapped URL segment missing OSC 8 hyperlink: %q", line)
			}
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

func TestFindURLAtTranscriptClickWrappedCommitURL(t *testing.T) {
	url := "https://github.com/Cod-e-Codes/marchat/commit/e579461ea4d75e8f4971bcf6095342f7fa2205e1"
	styles := getThemeStyles("patriot")
	msgs := []shared.Message{
		{
			Sender:    "Cody",
			Content:   url,
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
			MessageID: 84,
		},
	}
	const width = 62
	out := renderMessages(msgs, styles, "bob", []string{"Cody", "bob"}, width, true, true)
	rawLines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	clicked := false
	for relY, line := range rawLines {
		plain := plainTranscriptLine(line)
		if !strings.Contains(plain, "github.com") && !strings.Contains(plain, "e579461") {
			continue
		}
		lineWidth := ansi.StringWidth(plain)
		for relX := 0; relX <= lineWidth; relX++ {
			got := findURLAtTranscriptClick(rawLines, relY, relX)
			if got == "" {
				continue
			}
			if got != url {
				t.Fatalf("click relY=%d relX=%d: got %q want full URL", relY, relX, got)
			}
			clicked = true
		}
	}
	if !clicked {
		t.Fatal("no click position resolved the wrapped commit URL")
	}
}

func TestFindURLAtClickPositionThroughViewport(t *testing.T) {
	url := "https://github.com/Cod-e-Codes/marchat/commit/85bf012bde8a88b9730e9a4ff3015551556835a9"
	styles := getThemeStyles("patriot")
	msgs := []shared.Message{
		{
			Sender:    "Cody",
			Content:   url,
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
			MessageID: 84,
		},
	}
	const chatWidth = 62
	content := renderMessages(msgs, styles, "bob", []string{"Cody", "bob"}, chatWidth, true, true)
	lineURLs := buildTranscriptLineURLs(msgs, content)
	vp := viewport.New(viewport.WithWidth(chatWidth), viewport.WithHeight(20))
	vp.SetWidth(chatWidth)
	vp.SetHeight(20)
	vp.SetContent(content)

	m := &model{viewport: vp, transcriptLineURLs: lineURLs}
	x0, y0 := m.chatPanelOrigin()
	viewLines := strings.Split(strings.TrimRight(vp.View(), "\n"), "\n")

	clicked := false
	for relY, line := range viewLines {
		plain := plainTranscriptLine(line)
		if !strings.Contains(plain, "github.com") && !strings.Contains(plain, "85bf012") {
			continue
		}
		lineWidth := ansi.StringWidth(plain)
		for relX := 0; relX <= lineWidth; relX++ {
			got := m.findURLAtClickPosition(x0+relX, y0+relY)
			if got == "" {
				continue
			}
			if got != url {
				t.Fatalf("viewport click relY=%d relX=%d: got %q want full URL", relY, relX, got)
			}
			clicked = true
		}
	}
	if !clicked {
		t.Fatal("viewport path did not resolve wrapped commit URL")
	}
}

func TestExpandClickedURLFromMessage(t *testing.T) {
	full := "https://github.com/Cod-e-Codes/marchat/commit/85bf012bde8a88b9730e9a4ff3015551556835a9"
	msgs := []shared.Message{
		{Content: full, Type: shared.TextMessage, Channel: "general"},
	}
	got := expandClickedURL("https://github.com/Cod", msgs)
	if got != full {
		t.Fatalf("expand partial: got %q want %q", got, full)
	}
}

func TestChatPanelOriginIncludesBoxBorder(t *testing.T) {
	m := &model{viewport: viewport.New(viewport.WithWidth(62), viewport.WithHeight(10))}
	_, y0 := m.chatPanelOrigin()
	if y0 != 2 {
		t.Fatalf("y0=%d want 2 (header + chat box top border)", y0)
	}
	m.banner = "[OK] test"
	_, y0 = m.chatPanelOrigin()
	if y0 != 3 {
		t.Fatalf("y0 with banner=%d want 3", y0)
	}
}

func TestFindURLAtClickPositionExpandsPartialMatch(t *testing.T) {
	full := "https://github.com/Cod-e-Codes/marchat/commit/85bf012bde8a88b9730e9a4ff3015551556835a9"
	lineURLs := map[int][]string{0: {full}}
	m := &model{
		viewport:           viewport.New(viewport.WithWidth(80), viewport.WithHeight(5)),
		messages:           []shared.Message{{Content: full, Type: shared.TextMessage, Channel: "general"}},
		currentChannel:     "general",
		transcriptLineURLs: lineURLs,
	}
	m.viewport.SetContent("[20:20] Cody: https://github.com/Cod\n")
	x0, y0 := m.chatPanelOrigin()
	line := "[20:20] Cody: https://github.com/Cod"
	relX := strings.Index(line, "github.com/Cod") + len("github.com/")
	got := m.findURLAtClickPosition(x0+relX, y0)
	if got != full {
		t.Fatalf("partial viewport match: got %q want %q", got, full)
	}
}

func TestTranscriptLineURLsIndexWrappedCommitURL(t *testing.T) {
	url := "https://github.com/Cod-e-Codes/marchat/commit/85bf012bde8a88b9730e9a4ff3015551556835a9"
	styles := getThemeStyles("patriot")
	msgs := []shared.Message{{
		Sender: "Cody", Content: url, CreatedAt: time.Now(),
		Type: shared.TextMessage, MessageID: 84,
	}}
	lineURLs := buildTranscriptLineURLs(msgs, renderMessages(msgs, styles, "bob", []string{"Cody", "bob"}, 62, true, true))
	out := renderMessages(msgs, styles, "bob", []string{"Cody", "bob"}, 62, true, true)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	found := 0
	for i, line := range lines {
		if strings.Contains(plainTranscriptLine(line), "github.com") || strings.Contains(plainTranscriptLine(line), "85bf012") {
			got := lineURLs[i]
			if len(got) != 1 || got[0] != url {
				t.Fatalf("line %d urls=%v want [%q]", i, got, url)
			}
			found++
		}
	}
	if found < 2 {
		t.Fatalf("expected wrapped URL lines indexed, found %d", found)
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

func TestIsTranscriptSystemMessageNegativeClientNotice(t *testing.T) {
	msg := shared.Message{
		Sender:    "System",
		Content:   "Available themes: dark",
		MessageID: -1,
	}
	if !isTranscriptSystemMessage(msg) {
		t.Fatal("negative-id transcript notice should stay in transcript")
	}
	ephemeral := shared.Message{Sender: "System", Content: "Usage: :kick", MessageID: -2}
	if isTranscriptSystemMessage(ephemeral) {
		t.Fatal("negative-id usage line should be ephemeral")
	}
}
