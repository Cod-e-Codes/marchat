package main

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Cod-e-Codes/marchat/shared"
	"github.com/charmbracelet/x/ansi"
)

func TestBuildStatusFooter(t *testing.T) {
	tests := []struct {
		name            string
		connected       bool
		showHelp        bool
		unread          int
		useE2E          bool
		currentChannel  string
		activeDMThread  string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:            "connected_plain",
			connected:       true,
			showHelp:        false,
			unread:          0,
			useE2E:          false,
			currentChannel:  "general",
			activeDMThread:  "",
			wantContains:    []string{"Connected"},
			wantNotContains: []string{"Unread", "E2E", "#general", "Ctrl+H", "Unencrypted", "Msg info"},
		},
		{
			name:            "disconnected_shows_help",
			connected:       false,
			showHelp:        false,
			unread:          0,
			useE2E:          false,
			currentChannel:  "",
			activeDMThread:  "",
			wantContains:    []string{"Disconnected", "Press Ctrl+H for help"},
			wantNotContains: []string{"Msg info"},
		},
		{
			name:            "help_open_connected",
			connected:       true,
			showHelp:        true,
			unread:          0,
			useE2E:          false,
			currentChannel:  "general",
			activeDMThread:  "",
			wantContains:    []string{"Connected", "Press Ctrl+H to close help"},
			wantNotContains: []string{"Press Ctrl+H for help"},
		},
		{
			name:            "unread_e2e_channel",
			connected:       true,
			showHelp:        false,
			unread:          3,
			useE2E:          true,
			currentChannel:  "dev",
			activeDMThread:  "",
			wantContains:    []string{"Connected", "3 unread", "E2E", "#dev"},
			wantNotContains: []string{"Unencrypted", "Msg info"},
		},
		{
			name:            "dm_thread",
			connected:       true,
			showHelp:        false,
			unread:          0,
			useE2E:          false,
			currentChannel:  "general",
			activeDMThread:  "alice",
			wantContains:    []string{"Connected", "DM:alice"},
			wantNotContains: []string{"#general"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildStatusFooter(tt.connected, tt.showHelp, tt.unread, tt.useE2E, tt.currentChannel, tt.activeDMThread)
			for _, s := range tt.wantContains {
				if !strings.Contains(got, s) {
					t.Errorf("footer %q should contain %q", got, s)
				}
			}
			for _, s := range tt.wantNotContains {
				if strings.Contains(got, s) {
					t.Errorf("footer %q should not contain %q", got, s)
				}
			}
		})
	}
}

func TestStripKindForBanner(t *testing.T) {
	tests := []struct {
		text string
		want bannerStripKind
	}{
		{"", bannerStripInfo},
		{"[OK] Connected", bannerStripInfo},
		{"Msg info: full", bannerStripInfo},
		{"  [WARN] clipboard", bannerStripWarn},
		{"[WARN] Connection lost", bannerStripWarn},
		{"[ERROR] failed", bannerStripError},
		{"[ERROR] x [Sending...]", bannerStripError},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := stripKindForBanner(tt.text); got != tt.want {
				t.Fatalf("stripKindForBanner(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestLayoutBannerForStrip(t *testing.T) {
	long := "[ERROR] Failed to read file: open " + strings.Repeat("x", 200) + ": no such file"
	out := layoutBannerForStrip(long, 80)
	if strings.Contains(out, "\n") {
		t.Fatal("banner layout must be single line")
	}
	if len([]rune(out)) > 80 {
		t.Fatalf("expected truncation under width, got len %d", len([]rune(out)))
	}
	if !strings.HasSuffix(out, "...") {
		t.Fatal("expected ellipsis for long banner")
	}
}

func TestMaxMessageID(t *testing.T) {
	msgs := []shared.Message{
		{MessageID: 10},
		{MessageID: 2},
		{MessageID: 99},
	}
	if id := maxMessageID(msgs); id != 99 {
		t.Fatalf("maxMessageID = %d, want 99", id)
	}
	if id := maxMessageID(nil); id != 0 {
		t.Fatalf("maxMessageID(nil) = %d, want 0", id)
	}
}

func TestConfigureTextareaChrome(t *testing.T) {
	styles := getThemeStyles("modern")
	ta := textarea.New()
	before := ta.Styles().Focused.CursorLine.Render("x")
	configureTextareaChrome(&ta, styles.Input)
	after := ta.Styles().Focused.CursorLine.Render("x")
	if before == after {
		t.Fatal("expected CursorLine styling to change")
	}
	if !ta.Styles().Cursor.Blink {
		t.Fatal("expected cursor blink enabled")
	}
}

func TestNewMainTeaViewSetsTerminalBG(t *testing.T) {
	styles := getThemeStyles("modern")
	v := newMainTeaView(styles, "hello", false)
	if v.BackgroundColor == nil {
		t.Fatal("expected modern theme to set alt-screen background color")
	}
	if !v.AltScreen || v.MouseMode != tea.MouseModeCellMotion {
		t.Fatal("expected alt screen and mouse mode")
	}
}

func TestNewMainTeaViewShiftDisablesMouse(t *testing.T) {
	styles := getThemeStyles("modern")
	v := newMainTeaView(styles, "hello", true)
	if v.MouseMode != tea.MouseModeNone {
		t.Fatalf("shift-held view should disable mouse, got %v", v.MouseMode)
	}
}

func TestChromeComposerPanelFullWidth(t *testing.T) {
	styles := getThemeStyles("retro")
	row := chromeComposerPanel(styles, 72, 3, "type here", false)
	if lipgloss.Width(row) < 72 {
		t.Fatalf("expected full composer width, got %d", lipgloss.Width(row))
	}
}

func TestChromeComposerPanelPlaceholderFill(t *testing.T) {
	styles := getThemeStyles("modern")
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Prompt = "┃ "
	ta.SetWidth(composeInnerWidth(40))
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	configureTextareaChrome(&ta, styles.Input)

	panel := chromeComposerPanel(styles, 40, ta.Height(), ta.View(), true)
	lines := strings.Split(strings.TrimSuffix(panel, "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected 3 composer lines, got %d", len(lines))
	}
	inner := composeInnerWidth(40)
	for i, line := range lines {
		if w := lipgloss.Width(line); w < inner {
			t.Fatalf("line %d width %d < inner %d", i, w, inner)
		}
	}
}

func TestComposeInputWidth(t *testing.T) {
	if w := composeInputWidth(50); w != chromeFullWidth(50)-2 {
		t.Fatalf("composeInputWidth = %d, want %d", w, chromeFullWidth(50)-2)
	}
}

func TestChromeComposerPanelFillsRowBackground(t *testing.T) {
	styles := getThemeStyles("modern")
	content := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render("hi")
	panel := chromeComposerPanel(styles, 40, 1, content, false)
	if lipgloss.Width(panel) < 40 {
		t.Fatalf("composer width = %d, want >= 40", lipgloss.Width(panel))
	}
	lines := strings.Split(panel, "\n")
	if len(lines) == 0 {
		t.Fatal("expected at least one line")
	}
	if lipgloss.Width(lines[0]) < composeInnerWidth(40) {
		t.Fatalf("expected padded composer line width >= %d, got %d", composeInnerWidth(40), lipgloss.Width(lines[0]))
	}
}

func TestPadComposerLinesPreservesMultilineCursor(t *testing.T) {
	styles := getThemeStyles("modern")
	ta := textarea.New()
	ta.Prompt = "┃ "
	ta.SetWidth(composeInnerWidth(40))
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	configureTextareaChrome(&ta, styles.Input)
	ta.SetValue("hello\n")
	ta.Focus()

	raw := ta.View()
	withPlaceholder := padComposerLines(styles, composeInnerWidth(40), ta.Height(), raw, true)
	active := padComposerLines(styles, composeInnerWidth(40), ta.Height(), raw, false)
	if withPlaceholder == active {
		t.Fatal("placeholder flattening must not run while composing multiline text")
	}
	activeLines := strings.Split(strings.TrimSuffix(active, "\n"), "\n")
	if len(activeLines) < 2 {
		t.Fatal("expected multiline composer output")
	}
	if !strings.Contains(ansi.Strip(activeLines[1]), "┃") {
		t.Fatalf("expected cursor row content, got %q", ansi.Strip(activeLines[1]))
	}
}

func TestComposerLineIsBareBuffer(t *testing.T) {
	if !composerLineIsBareBuffer("┃ ") {
		t.Fatal("expected bare buffer line")
	}
	if composerLineIsBareBuffer("┃ hello") {
		t.Fatal("expected non-buffer line")
	}
}

func TestChromeComposerPanelPlaceholderBufferRowsSolid(t *testing.T) {
	styles := getThemeStyles("modern")
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Prompt = "┃ "
	ta.SetWidth(composeInnerWidth(40))
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	configureTextareaChrome(&ta, styles.Input)

	panel := chromeComposerPanel(styles, 40, ta.Height(), ta.View(), true)
	lines := strings.Split(strings.TrimSuffix(panel, "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	inner := composeInnerWidth(40)
	for i := 1; i < 3; i++ {
		if strings.TrimSpace(ansi.Strip(lines[i])) != "" {
			t.Fatalf("line %d should be solid fill, plain=%q", i, ansi.Strip(lines[i]))
		}
		if lipgloss.Width(lines[i]) < inner {
			t.Fatalf("line %d width %d < %d", i, lipgloss.Width(lines[i]), inner)
		}
	}
}
