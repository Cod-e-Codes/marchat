package main

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

// overlayCapturesKeyboard is true when a full-screen overlay owns keyboard input
// besides scroll and dismiss keys (help, admin DB menu).
func (m *model) overlayCapturesKeyboard() bool {
	return m.showHelp || m.showDBMenu
}

// subModelCapturesInput is true when file picker or code snippet modals are open.
func (m *model) subModelCapturesInput() bool {
	return m.showFilePicker || m.showCodeSnippet
}

// textareaWantsArrowKeys is true when up/down should move the cursor inside a
// multiline composer instead of scrolling the chat transcript.
func (m *model) textareaWantsArrowKeys() bool {
	return m.textarea.Focused() && strings.Contains(m.textarea.Value(), "\n")
}

// handleComposerScrollKey routes arrow/page keys between the composer and the
// active scroll viewport. Returns handled and an optional command.
func (m *model) handleComposerScrollKey(v tea.KeyPressMsg) (bool, tea.Cmd) {
	if m.textareaWantsArrowKeys() {
		switch {
		case key.Matches(v, m.keys.ScrollUp), key.Matches(v, m.keys.ScrollDown):
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(v)
			return true, cmd
		}
	}
	switch {
	case key.Matches(v, m.keys.ScrollUp):
		m.scrollActiveViewport(-1)
		return true, nil
	case key.Matches(v, m.keys.ScrollDown):
		m.scrollActiveViewport(1)
		if cmd := m.maybeFlushReadReceipt(); cmd != nil {
			return true, cmd
		}
		return true, nil
	case key.Matches(v, m.keys.PageUp):
		m.pageScrollActiveViewport(-1)
		return true, nil
	case key.Matches(v, m.keys.PageDown):
		m.pageScrollActiveViewport(1)
		if cmd := m.maybeFlushReadReceipt(); cmd != nil {
			return true, cmd
		}
		return true, nil
	default:
		return false, nil
	}
}

// activeScrollViewport returns the viewport that should receive scroll and wheel
// input for the current UI mode. Nil when a sub-model owns scrolling.
func (m *model) activeScrollViewport() *viewport.Model {
	switch {
	case m.showHelp:
		return &m.helpViewport
	case m.showDBMenu:
		return &m.dbMenuViewport
	case m.showCodeSnippet, m.showFilePicker:
		return nil
	case m.textarea.Focused():
		return &m.viewport
	default:
		return &m.userListViewport
	}
}

func (m *model) scrollActiveViewport(lines int) {
	vp := m.activeScrollViewport()
	if vp == nil || lines == 0 {
		return
	}
	if lines > 0 {
		vp.ScrollDown(lines)
	} else {
		vp.ScrollUp(-lines)
	}
}

// maybeFlushReadReceipt clears unread and schedules a read receipt only after the
// chat transcript viewport was scrolled to the tail (not help/DB menu/user list).
func (m *model) maybeFlushReadReceipt() tea.Cmd {
	if m.activeScrollViewport() != &m.viewport || !m.viewport.AtBottom() {
		return nil
	}
	m.unreadCount = 0
	return m.scheduleReadReceiptFlush()
}

func (m *model) pageScrollActiveViewport(direction int) {
	vp := m.activeScrollViewport()
	if vp == nil {
		return
	}
	h := vp.Height()
	if h < 1 {
		h = 1
	}
	if direction > 0 {
		vp.ScrollDown(h)
	} else {
		vp.ScrollUp(h)
	}
}

func (m *model) updateActiveScrollViewport(msg tea.Msg) bool {
	vp := m.activeScrollViewport()
	if vp == nil {
		return false
	}
	updated, _ := vp.Update(msg)
	*vp = updated
	return true
}

func helpViewportContentHeight(totalHeight int) int {
	h := totalHeight - 3 // footer border + padding + text
	if h < 10 {
		return 10
	}
	return h
}

func dbMenuViewportDimensions(totalWidth, totalHeight int) (width, height int) {
	width = 60
	height = 15
	if totalWidth < width+4 {
		width = totalWidth - 4
	}
	if totalHeight < height+4 {
		height = totalHeight - 4
	}
	if width < 20 {
		width = 20
	}
	if height < 5 {
		height = 5
	}
	return width, height
}

func applyMouseWheelToList(l *list.Model, msg tea.MouseWheelMsg) {
	if l == nil {
		return
	}
	switch msg.Button {
	case tea.MouseWheelDown:
		l.CursorDown()
	case tea.MouseWheelUp:
		l.CursorUp()
	}
}
