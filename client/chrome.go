// Chrome: footer vs banner rules for the main chat TUI.
//
// Footer shows stable connection state and a few predictable segments (unread
// count, optional E2E label, optional non-default channel). Do not put
// timer-driven or one-off strings in the footer.
//
// Banner strip above the transcript uses themeStyles banner strip styles:
// error, warn, and info bands keyed by [ERROR], [WARN], and everything else.
// Keep one surface per concern so users are not confused by duplicate or
// vanishing status text.
package main

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Cod-e-Codes/marchat/shared"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucasb-eyer/go-colorful"
)

const readReceiptDebounce = 750 * time.Millisecond

// altScreenFill is the Bubble Tea v2 alt-screen base color (unpainted cells). Kept black
// so the transcript reads on true black; header/footer/input carry theme chrome colors.
const altScreenFill = "#000000"

// transcriptFill is the chat viewport interior behind messages.
const transcriptFill = "#000000"

// chromeFullWidth matches header and footer: chat viewport plus user list plus gap.
func chromeFullWidth(viewportW int) int {
	return viewportW + userListWidth + 4
}

// composeInputWidth is the bubbles textarea width inside the full-width composer bar
// (matches horizontal padding in chromeComposerPanel).
func composeInputWidth(viewportW int) int {
	return composeInnerWidth(chromeFullWidth(viewportW))
}

func composeInnerWidth(fullW int) int {
	w := fullW - 2
	if w < 20 {
		return 20
	}
	return w
}

// lineStyleFromInput builds per-line textarea styles that carry the composer fill.
func lineStyleFromInput(input lipgloss.Style) lipgloss.Style {
	s := lipgloss.NewStyle()
	if bg := input.GetBackground(); bg != nil {
		s = s.Background(bg)
	}
	if fg := input.GetForeground(); fg != nil {
		s = s.Foreground(fg)
	}
	return s
}

// configureTextareaChrome syncs textarea with theme Input. Composer chrome paints the
// full row background; textarea line styles carry fill for trailing pad spaces.
func configureTextareaChrome(ta *textarea.Model, input lipgloss.Style) {
	s := textarea.DefaultDarkStyles()
	line := lineStyleFromInput(input)
	faint := line.Faint(true)
	for _, state := range []*textarea.StyleState{&s.Focused, &s.Blurred} {
		state.Base = lipgloss.NewStyle()
		state.Text = line
		state.CursorLine = line
		state.Prompt = faint
		state.Placeholder = faint
		state.LineNumber = faint
		state.CursorLineNumber = faint
		state.EndOfBuffer = line
	}
	s.Cursor.Blink = true
	if fg := input.GetForeground(); fg != nil {
		s.Cursor.Color = fg
	}
	ta.SetStyles(s)
	ta.SetVirtualCursor(true)
}

// composerLineIsBareBuffer reports placeholder end-of-buffer rows (prompt only).
func composerLineIsBareBuffer(line string) bool {
	plain := strings.TrimSpace(ansi.Strip(line))
	plain = strings.TrimPrefix(plain, "┃")
	plain = strings.TrimSpace(plain)
	return plain == "" || len([]rune(plain)) <= 1
}

// padComposerLines extends every row to innerW and paints the full row with the input fill.
// Bubbles textarea placeholder buffer rows only style the prompt plus one end-of-buffer rune;
// the viewport may pad with unstyled spaces that read as alt-screen black.
func padComposerLines(styles themeStyles, innerW, minLines int, content string, placeholderMode bool) string {
	lines := strings.Split(content, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	fill := styles.Input
	for len(lines) < minLines {
		lines = append(lines, "")
	}
	if len(lines) > minLines {
		lines = lines[:minLines]
	}
	for i := range lines {
		// Only flatten empty buffer rows before typing; multiline cursor rows look bare too.
		if placeholderMode && i > 0 && composerLineIsBareBuffer(lines[i]) {
			lines[i] = fill.Width(innerW).Render("")
			continue
		}
		placed := lipgloss.PlaceHorizontal(innerW, lipgloss.Left, lines[i],
			lipgloss.WithWhitespaceStyle(fill))
		lines[i] = fill.Width(innerW).Render(placed)
	}
	return strings.Join(lines, "\n")
}

// chromeComposerPanel renders the full-width message composer (header/footer alignment).
func chromeComposerPanel(styles themeStyles, fullW, minLines int, inputContent string, placeholderMode bool) string {
	innerW := composeInnerWidth(fullW)
	if minLines < 1 {
		minLines = 1
	}
	padded := padComposerLines(styles, innerW, minLines, inputContent, placeholderMode)
	return styles.Input.Width(fullW).Padding(0, 1).Render(padded)
}

// chromeTypingLine renders a full-width typing indicator under the main grid.
func chromeTypingLine(fullW int, line string) string {
	if strings.TrimSpace(line) == "" {
		return ""
	}
	return lipgloss.NewStyle().Faint(true).Italic(true).Width(fullW).PaddingLeft(1).Render(line)
}

// newMainTeaView renders the main TUI on the alt screen. Bubble Tea v2 leaves unpainted
// alt-screen cells black unless BackgroundColor is set; lipgloss Background.Render alone
// only covers laid-out content (main v1 did not have this gap).
func newMainTeaView(styles themeStyles, ui string, shiftHeld bool) tea.View {
	v := tea.NewView(styles.Background.Render(ui))
	if bg, ok := styles.terminalBGColor(); ok {
		v.BackgroundColor = bg
	}
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	if shiftHeld {
		// Release mouse capture so the terminal can drag-select transcript text (Shift+drag).
		v.MouseMode = tea.MouseModeNone
	}
	return v
}

func (s themeStyles) terminalBGColor() (color.Color, bool) {
	if s.screenBG == "" {
		return nil, false
	}
	c, err := colorful.Hex(s.screenBG)
	if err != nil {
		return nil, false
	}
	return c, true
}

// updateModifierKeys tracks Shift for terminal text selection passthrough.
func (m *model) updateModifierKeys(k tea.Key) {
	switch k.Code {
	case tea.KeyLeftShift, tea.KeyRightShift:
		m.shiftHeld = true
	default:
		m.shiftHeld = k.Mod&tea.ModShift != 0
	}
}

// layoutBannerForStrip collapses newlines to spaces and truncates to one line so a
// long [ERROR] path does not consume most of the terminal height under JoinVertical.
func layoutBannerForStrip(text string, width int) string {
	if width < 24 {
		width = 24
	}
	s := strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	s = strings.Join(strings.Fields(s), " ")
	max := width - 3
	if max < 16 {
		max = 16
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-3]) + "..."
}

// bannerStripKind selects full-width banner strip colors in View.
type bannerStripKind int

const (
	bannerStripInfo bannerStripKind = iota
	bannerStripWarn
	bannerStripError
)

// stripKindForBanner maps banner text (including optional " [Sending...]" suffix)
// to a strip kind. Prefixes are case-sensitive to match existing banner strings.
func stripKindForBanner(bannerText string) bannerStripKind {
	t := strings.TrimSpace(bannerText)
	if strings.HasPrefix(t, "[ERROR]") {
		return bannerStripError
	}
	if strings.HasPrefix(t, "[WARN]") {
		return bannerStripWarn
	}
	return bannerStripInfo
}

// BannerStrip returns the lipgloss style for the full-width status strip.
func (s themeStyles) BannerStrip(kind bannerStripKind) lipgloss.Style {
	switch kind {
	case bannerStripError:
		return s.BannerError
	case bannerStripWarn:
		return s.BannerWarn
	default:
		return s.BannerInfo
	}
}

// buildStatusFooter returns a single footer line: connection, optional unread,
// optional E2E when enabled, optional channel when not general, and help text
// only when disconnected or when help overlay is open (stable while open).
func buildStatusFooter(connected, showHelp bool, unread int, useE2E bool, currentChannel, activeDMThread string) string {
	var parts []string
	if connected {
		parts = append(parts, "Connected")
	} else {
		parts = append(parts, "Disconnected")
	}
	if unread > 0 {
		parts = append(parts, fmt.Sprintf("%d unread", unread))
	}
	if useE2E {
		parts = append(parts, "E2E")
	}
	ch := strings.TrimSpace(strings.ToLower(currentChannel))
	if ch != "" && ch != "general" {
		parts = append(parts, "#"+ch)
	}
	if dm := strings.TrimSpace(activeDMThread); dm != "" {
		parts = append(parts, "DM:"+dm)
	}
	if showHelp && connected {
		parts = append(parts, "Press Ctrl+H to close help")
	} else if !connected {
		parts = append(parts, "Press Ctrl+H for help")
	}
	return strings.Join(parts, " | ")
}

// maxMessageID returns the largest message_id in the transcript, or 0 if none.
func maxMessageID(msgs []shared.Message) int64 {
	var max int64
	for i := range msgs {
		if msgs[i].MessageID > max {
			max = msgs[i].MessageID
		}
	}
	return max
}

// scheduleReadReceiptFlush debounces outbound read_receipt while the viewport
// is pinned to the tail. Coalesces bursts into one send after readReceiptDebounce.
func (m *model) scheduleReadReceiptFlush() tea.Cmd {
	if m.conn == nil || !m.connected || !m.viewport.AtBottom() {
		return nil
	}
	maxID := maxMessageID(m.messages)
	if maxID == 0 || maxID <= m.lastReadReceiptSentID {
		return nil
	}
	if m.readReceiptFlushScheduled {
		return nil
	}
	m.readReceiptFlushScheduled = true
	return tea.Tick(readReceiptDebounce, func(time.Time) tea.Msg {
		return readReceiptFlushMsg{}
	})
}

// flushReadReceipt sends a single read_receipt for the latest message id at
// the tail. On failure, sets banner and leaves lastReadReceiptSentID unchanged.
func (m *model) flushReadReceipt() tea.Cmd {
	if m.conn == nil || !m.connected || !m.viewport.AtBottom() {
		return nil
	}
	maxID := maxMessageID(m.messages)
	if maxID == 0 || maxID <= m.lastReadReceiptSentID {
		return nil
	}
	out := shared.Message{
		Type:      shared.ReadReceiptType,
		Sender:    m.cfg.Username,
		MessageID: maxID,
		Channel:   normalizeChannel(m.currentChannel),
	}
	if err := m.conn.WriteJSON(out); err != nil {
		m.banner = "[ERROR] Failed to send read receipt: " + err.Error()
		return nil
	}
	m.lastReadReceiptSentID = maxID
	if mid := maxMessageID(m.messages); mid > m.lastReadReceiptSentID && m.viewport.AtBottom() {
		return m.scheduleReadReceiptFlush()
	}
	return nil
}
