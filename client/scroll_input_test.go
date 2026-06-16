package main

import (
	"testing"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

func TestHelpViewportContentHeight(t *testing.T) {
	tests := []struct {
		total, want int
	}{
		{20, 17},
		{12, 10},
		{5, 10},
	}
	for _, tt := range tests {
		if got := helpViewportContentHeight(tt.total); got != tt.want {
			t.Fatalf("helpViewportContentHeight(%d) = %d, want %d", tt.total, got, tt.want)
		}
	}
}

func TestDBMenuViewportDimensions(t *testing.T) {
	w, h := dbMenuViewportDimensions(120, 40)
	if w != 60 || h != 15 {
		t.Fatalf("default db menu = %dx%d, want 60x15", w, h)
	}
	w, h = dbMenuViewportDimensions(50, 12)
	if w != 46 || h != 8 {
		t.Fatalf("small terminal db menu = %dx%d, want 46x8", w, h)
	}
}

func TestActiveScrollViewport(t *testing.T) {
	base := func() *model {
		return &model{
			textarea: textarea.New(),
		}
	}

	m := base()
	m.showHelp = true
	if m.activeScrollViewport() != &m.helpViewport {
		t.Fatal("expected help viewport when help is open")
	}

	m = base()
	m.showDBMenu = true
	if m.activeScrollViewport() != &m.dbMenuViewport {
		t.Fatal("expected db menu viewport when db menu is open")
	}

	m = base()
	m.showHelp = true
	m.showDBMenu = true
	if m.activeScrollViewport() != &m.helpViewport {
		t.Fatal("help should take priority over db menu")
	}

	m = base()
	m.textarea.Focus()
	if m.activeScrollViewport() != &m.viewport {
		t.Fatal("expected chat viewport when textarea focused")
	}

	m = base()
	if m.activeScrollViewport() != &m.userListViewport {
		t.Fatal("expected user list viewport by default")
	}

	m = base()
	m.showFilePicker = true
	if m.activeScrollViewport() != nil {
		t.Fatal("expected nil when file picker owns scroll")
	}
}

func TestUpdateActiveScrollViewportMouseWheel(t *testing.T) {
	m := &model{
		showHelp:     true,
		helpViewport: viewport.New(viewport.WithWidth(40), viewport.WithHeight(5)),
	}
	long := ""
	for i := 0; i < 40; i++ {
		long += "line\n"
	}
	m.helpViewport.SetContent(long)

	if !m.updateActiveScrollViewport(tea.MouseWheelMsg{Button: tea.MouseWheelDown}) {
		t.Fatal("expected wheel to be handled")
	}
	if m.helpViewport.YOffset() == 0 {
		t.Fatal("expected help viewport to scroll down on wheel")
	}

	start := m.helpViewport.YOffset()
	if !m.updateActiveScrollViewport(tea.MouseWheelMsg{Button: tea.MouseWheelUp}) {
		t.Fatal("expected wheel up to be handled")
	}
	if m.helpViewport.YOffset() >= start {
		t.Fatalf("expected scroll up, yoffset=%d start=%d", m.helpViewport.YOffset(), start)
	}
}

func TestApplyMouseWheelToList(t *testing.T) {
	// Covered indirectly via file_picker tests; smoke-test helper does not panic.
	var l list.Model
	applyMouseWheelToList(&l, tea.MouseWheelMsg{Button: tea.MouseWheelDown})
}

func TestOverlayCapturesKeyboard(t *testing.T) {
	m := &model{}
	if m.overlayCapturesKeyboard() {
		t.Fatal("expected false by default")
	}
	m.showHelp = true
	if !m.overlayCapturesKeyboard() {
		t.Fatal("help should capture keyboard")
	}
	m.showHelp = false
	m.showDBMenu = true
	if !m.overlayCapturesKeyboard() {
		t.Fatal("db menu should capture keyboard")
	}
}

func TestSubModelCapturesInput(t *testing.T) {
	m := &model{}
	if m.subModelCapturesInput() {
		t.Fatal("expected false by default")
	}
	m.showFilePicker = true
	if !m.subModelCapturesInput() {
		t.Fatal("file picker should capture input")
	}
	m.showFilePicker = false
	m.showCodeSnippet = true
	if !m.subModelCapturesInput() {
		t.Fatal("code snippet should capture input")
	}
}

func TestMaybeFlushReadReceipt_ScopedToChatViewport(t *testing.T) {
	m := &model{textarea: textarea.New()}
	m.textarea.Focus()
	m.unreadCount = 5

	m.showHelp = true
	if cmd := m.maybeFlushReadReceipt(); cmd != nil {
		t.Fatal("read receipt must not flush while help is open")
	}
	if m.unreadCount != 5 {
		t.Fatalf("unread count should stay %d while help blocks flush, got %d", 5, m.unreadCount)
	}

	m.showHelp = false
	m.showDBMenu = true
	if cmd := m.maybeFlushReadReceipt(); cmd != nil {
		t.Fatal("read receipt must not flush while db menu is open")
	}
	if m.unreadCount != 5 {
		t.Fatalf("unread count should stay %d while db menu blocks flush, got %d", 5, m.unreadCount)
	}

	m.showDBMenu = false
	m.textarea.Blur()
	if cmd := m.maybeFlushReadReceipt(); cmd != nil {
		t.Fatal("read receipt must not flush when user list is the active scroll target")
	}

	m.textarea.Focus()
	if cmd := m.maybeFlushReadReceipt(); cmd != nil {
		t.Fatal("read receipt must not flush when chat viewport is not at bottom")
	}
}
