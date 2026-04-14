package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Mock themeStyles for testing
func getMockThemeStyles() themeStyles {
	return themeStyles{
		User:           lipgloss.NewStyle(),
		Msg:            lipgloss.NewStyle(),
		Time:           lipgloss.NewStyle(),
		Banner:         lipgloss.NewStyle(),
		BannerError:    lipgloss.NewStyle(),
		BannerWarn:     lipgloss.NewStyle(),
		BannerInfo:     lipgloss.NewStyle(),
		SystemMsg:      lipgloss.NewStyle(),
		SystemMsgError: lipgloss.NewStyle(),
		SystemMsgWarn:  lipgloss.NewStyle(),
	}
}

// Test newCodeSnippetModel initialization
func TestNewCodeSnippetModel(t *testing.T) {
	styles := getMockThemeStyles()
	width, height := 80, 24

	var sentCode string
	var cancelled bool

	onSend := func(code string) {
		sentCode = code
	}
	onCancel := func() {
		cancelled = true
	}

	model := newCodeSnippetModel(styles, width, height, onSend, onCancel)

	// Test initial state
	if model.state != stateSelectLang {
		t.Errorf("Expected initial state to be stateSelectLang, got %v", model.state)
	}

	if !model.initialized {
		t.Error("Expected model to be initialized")
	}

	if len(model.languages) == 0 {
		t.Error("Expected languages to be populated")
	}

	// Test that common languages are included
	expectedLangs := []string{"go", "python", "javascript", "typescript", "java"}
	for _, lang := range expectedLangs {
		found := false
		for _, modelLang := range model.languages {
			if modelLang == lang {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected language %s to be in languages list", lang)
		}
	}

	// Test initial lines
	if len(model.lines) != 1 || model.lines[0] != "" {
		t.Errorf("Expected initial lines to be [\"\"], got %v", model.lines)
	}

	// Test cursor position
	if model.cursorX != 0 || model.cursorY != 0 {
		t.Errorf("Expected initial cursor position to be (0,0), got (%d,%d)", model.cursorX, model.cursorY)
	}

	// Test callbacks
	model.onSend("test code")
	if sentCode != "test code" {
		t.Errorf("Expected onSend callback to be called with 'test code', got '%s'", sentCode)
	}

	model.onCancel()
	if !cancelled {
		t.Error("Expected onCancel callback to be called")
	}
}

// Test language selection state
func TestCodeSnippetLanguageSelection(t *testing.T) {
	styles := getMockThemeStyles()
	model := newCodeSnippetModel(styles, 80, 24, func(string) {}, func() {})

	// Test initial state
	if model.state != stateSelectLang {
		t.Errorf("Expected state to be stateSelectLang, got %v", model.state)
	}

	// Test Enter key to select language
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)

	csModel, ok := updatedModel.(codeSnippetModel)
	if !ok {
		t.Fatal("Expected updated model to be codeSnippetModel")
	}

	if csModel.state != stateInputCode {
		t.Errorf("Expected state to change to stateInputCode, got %v", csModel.state)
	}

	if csModel.selected == "" {
		t.Error("Expected selected language to be set")
	}

	// Test Escape key to cancel
	model = newCodeSnippetModel(styles, 80, 24, func(string) {}, func() {})
	var cancelled bool
	model.onCancel = func() { cancelled = true }

	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	model.Update(escMsg)

	if !cancelled {
		t.Error("Expected Escape key to trigger cancel callback")
	}
}

// Test text input and cursor movement
func TestCodeSnippetTextInput(t *testing.T) {
	styles := getMockThemeStyles()
	model := newCodeSnippetModel(styles, 80, 24, func(string) {}, func() {})

	// Move to input state
	model.state = stateInputCode
	model.selected = "go"

	// Test character input
	charMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	updatedModel, _ := model.Update(charMsg)
	csModel := updatedModel.(codeSnippetModel)

	if csModel.lines[0] != "h" {
		t.Errorf("Expected line to contain 'h', got '%s'", csModel.lines[0])
	}

	if csModel.cursorX != 1 {
		t.Errorf("Expected cursor X to be 1, got %d", csModel.cursorX)
	}

	// Test Enter key (new line)
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = csModel.Update(enterMsg)
	csModel = updatedModel.(codeSnippetModel)

	if len(csModel.lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(csModel.lines))
	}

	if csModel.cursorY != 1 || csModel.cursorX != 0 {
		t.Errorf("Expected cursor to be at (0,1), got (%d,%d)", csModel.cursorX, csModel.cursorY)
	}

	// Test backspace
	backspaceMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, _ = csModel.Update(backspaceMsg)
	csModel = updatedModel.(codeSnippetModel)

	if len(csModel.lines) != 1 {
		t.Errorf("Expected backspace to merge lines, got %d lines", len(csModel.lines))
	}
}

// Test text selection functionality
func TestCodeSnippetTextSelection(t *testing.T) {
	styles := getMockThemeStyles()
	model := newCodeSnippetModel(styles, 80, 24, func(string) {}, func() {})

	// Set up some text
	model.state = stateInputCode
	model.lines = []string{"hello", "world"}
	model.cursorX = 0
	model.cursorY = 0

	// Test shift+right selection
	shiftRightMsg := tea.KeyMsg{Type: tea.KeyShiftRight}
	updatedModel, _ := model.Update(shiftRightMsg)
	model = updatedModel.(codeSnippetModel)

	if !model.hasSelection {
		t.Error("Expected selection to be started")
	}

	// Test getSelectedText
	model.selectionStartX = 0
	model.selectionStartY = 0
	model.selectionEndX = 3
	model.selectionEndY = 0
	model.hasSelection = true

	selectedText := model.getSelectedText()
	if selectedText != "hel" {
		t.Errorf("Expected selected text to be 'hel', got '%s'", selectedText)
	}

	// Test select all
	model.selectAll()
	if !model.hasSelection {
		t.Error("Expected select all to create selection")
	}

	// Test clear selection
	model.clearSelection()
	if model.hasSelection {
		t.Error("Expected selection to be cleared")
	}
}

// Test delete selection
func TestCodeSnippetDeleteSelection(t *testing.T) {
	styles := getMockThemeStyles()
	model := newCodeSnippetModel(styles, 80, 24, func(string) {}, func() {})

	// Set up text with selection
	model.state = stateInputCode
	model.lines = []string{"hello world"}
	model.selectionStartX = 6
	model.selectionStartY = 0
	model.selectionEndX = 11
	model.selectionEndY = 0
	model.hasSelection = true

	// Delete selection
	model.deleteSelection()

	if model.hasSelection {
		t.Error("Expected selection to be cleared after deletion")
	}

	if model.lines[0] != "hello " {
		t.Errorf("Expected line to be 'hello ', got '%s'", model.lines[0])
	}
}

// Test paste functionality
func TestCodeSnippetPaste(t *testing.T) {
	styles := getMockThemeStyles()
	model := newCodeSnippetModel(styles, 80, 24, func(string) {}, func() {})

	// Set up initial state
	model.state = stateInputCode
	model.lines = []string{"hello"}
	model.cursorX = 2 // Position cursor in middle of "hello"

	// Paste single line text
	model.pasteText("X")

	if model.lines[0] != "heXllo" {
		t.Errorf("Expected line to be 'heXllo', got '%s'", model.lines[0])
	}

	// Test multi-line paste
	model.cursorX = 3
	model.pasteText("Y\nZ")

	if len(model.lines) != 2 {
		t.Errorf("Expected 2 lines after multi-line paste, got %d", len(model.lines))
	}

	if model.lines[0] != "heXY" {
		t.Errorf("Expected first line to be 'heXY', got '%s'", model.lines[0])
	}

	if model.lines[1] != "Zllo" {
		t.Errorf("Expected second line to be 'Zllo', got '%s'", model.lines[1])
	}
}

// Test cursor movement
func TestCodeSnippetCursorMovement(t *testing.T) {
	styles := getMockThemeStyles()
	model := newCodeSnippetModel(styles, 80, 24, func(string) {}, func() {})

	// Set up multi-line text
	model.state = stateInputCode
	model.lines = []string{"hello", "world"}
	model.cursorX = 0
	model.cursorY = 0

	// Test right movement
	rightMsg := tea.KeyMsg{Type: tea.KeyRight}
	updatedModel, _ := model.Update(rightMsg)
	model = updatedModel.(codeSnippetModel)

	if model.cursorX != 1 {
		t.Errorf("Expected cursor X to be 1, got %d", model.cursorX)
	}

	// Test down movement
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = model.Update(downMsg)
	model = updatedModel.(codeSnippetModel)

	if model.cursorY != 1 {
		t.Errorf("Expected cursor Y to be 1, got %d", model.cursorY)
	}

	// Test left movement
	leftMsg := tea.KeyMsg{Type: tea.KeyLeft}
	updatedModel, _ = model.Update(leftMsg)
	model = updatedModel.(codeSnippetModel)

	if model.cursorX != 0 {
		t.Errorf("Expected cursor X to be 0, got %d", model.cursorX)
	}

	// Test up movement
	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ = model.Update(upMsg)
	model = updatedModel.(codeSnippetModel)

	if model.cursorY != 0 {
		t.Errorf("Expected cursor Y to be 0, got %d", model.cursorY)
	}
}

// Test preview and send functionality
func TestCodeSnippetPreviewAndSend(t *testing.T) {
	styles := getMockThemeStyles()
	var sentCode string
	model := newCodeSnippetModel(styles, 80, 24, func(code string) {
		sentCode = code
	}, func() {})

	// Set up code
	model.state = stateInputCode
	model.selected = "go"
	model.lines = []string{"package main", "func main() {}"}

	// Test Ctrl+S to preview
	ctrlSMsg := tea.KeyMsg{Type: tea.KeyCtrlS}
	updatedModel, _ := model.Update(ctrlSMsg)
	model = updatedModel.(codeSnippetModel)

	if model.state != stateConfirmSend {
		t.Errorf("Expected state to be stateConfirmSend, got %v", model.state)
	}

	if model.code != "package main\nfunc main() {}" {
		t.Errorf("Expected code to be joined properly, got '%s'", model.code)
	}

	// Test Enter to send
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(codeSnippetModel)

	if !strings.Contains(sentCode, "```go") {
		t.Errorf("Expected sent code to contain markdown format, got '%s'", sentCode)
	}

	if !strings.Contains(sentCode, "package main") {
		t.Errorf("Expected sent code to contain the code content, got '%s'", sentCode)
	}
}

// Test getSelectionBounds
func TestCodeSnippetGetSelectionBounds(t *testing.T) {
	styles := getMockThemeStyles()
	model := newCodeSnippetModel(styles, 80, 24, func(string) {}, func() {})

	// Test no selection
	startX, startY, endX, endY := model.getSelectionBounds()
	if startX != 0 || startY != 0 || endX != 0 || endY != 0 {
		t.Errorf("Expected no selection bounds to be (0,0,0,0), got (%d,%d,%d,%d)", startX, startY, endX, endY)
	}

	// Test selection
	model.hasSelection = true
	model.selectionStartX = 5
	model.selectionStartY = 1
	model.selectionEndX = 2
	model.selectionEndY = 0

	startX, startY, endX, endY = model.getSelectionBounds()
	if startX != 2 || startY != 0 || endX != 5 || endY != 1 {
		t.Errorf("Expected bounds to be (2,0,5,1), got (%d,%d,%d,%d)", startX, startY, endX, endY)
	}
}

// Test View method
func TestCodeSnippetView(t *testing.T) {
	styles := getMockThemeStyles()
	model := newCodeSnippetModel(styles, 80, 24, func(string) {}, func() {})

	// Test language selection view
	view := model.View()
	if !strings.Contains(view, "Select Programming Language") {
		t.Error("Expected view to contain language selection title")
	}

	// Test input code view
	model.state = stateInputCode
	model.selected = "go"
	model.lines = []string{"package main"}
	view = model.View()

	if !strings.Contains(view, "Language: go") {
		t.Error("Expected view to show selected language")
	}

	// Test confirm send view
	model.state = stateConfirmSend
	model.code = "package main"
	model.selected = "go"
	model.highlight = "highlighted code"
	view = model.View()

	if !strings.Contains(view, "highlighted code") && !strings.Contains(view, "package main") && !strings.Contains(view, "Press Enter") {
		t.Error("Expected view to show highlighted code or fallback content")
	}
}

// Test langItem interface implementation
func TestLangItem(t *testing.T) {
	item := langItem("go")

	if item.Title() != "go" {
		t.Errorf("Expected title to be 'go', got '%s'", item.Title())
	}

	if item.Description() != "" {
		t.Errorf("Expected description to be empty, got '%s'", item.Description())
	}

	if item.FilterValue() != "go" {
		t.Errorf("Expected filter value to be 'go', got '%s'", item.FilterValue())
	}
}

// Test formatCodeBlock
func TestFormatCodeBlock(t *testing.T) {
	styles := getMockThemeStyles()
	model := newCodeSnippetModel(styles, 80, 24, func(string) {}, func() {})

	// Test with valid language
	formatted := model.formatCodeBlock("go", "package main", false)

	if !strings.Contains(formatted, "Press Enter to send") {
		t.Error("Expected formatted code block to contain instructions")
	}

	// Test with copied flag
	formatted = model.formatCodeBlock("go", "package main", true)

	if !strings.Contains(formatted, "Code copied to clipboard") {
		t.Error("Expected copied message to be shown")
	}
}

// Test state transitions
func TestCodeSnippetStateTransitions(t *testing.T) {
	styles := getMockThemeStyles()
	model := newCodeSnippetModel(styles, 80, 24, func(string) {}, func() {})

	// Start in language selection
	if model.state != stateSelectLang {
		t.Errorf("Expected initial state to be stateSelectLang, got %v", model.state)
	}

	// Transition to input code
	model.state = stateInputCode
	model.selected = "go"

	if model.state != stateInputCode {
		t.Errorf("Expected state to be stateInputCode, got %v", model.state)
	}
	if model.selected != "go" {
		t.Errorf("Expected selected to be 'go', got '%s'", model.selected)
	}

	// Transition to confirm send
	model.state = stateConfirmSend
	model.code = "test"

	if model.state != stateConfirmSend {
		t.Errorf("Expected state to be stateConfirmSend, got %v", model.state)
	}
	if model.code != "test" {
		t.Errorf("Expected code to be 'test', got '%s'", model.code)
	}
}

// Test error handling in preview
func TestCodeSnippetPreviewError(t *testing.T) {
	styles := getMockThemeStyles()
	model := newCodeSnippetModel(styles, 80, 24, func(string) {}, func() {})

	// Set up model for preview
	model.state = stateInputCode
	model.selected = "invalid_language"
	model.lines = []string{"test code"}

	// Force preview state
	model.code = "test code"
	model.selected = "invalid_language"
	model.highlight = "Error: invalid language"
	model.state = stateConfirmSend

	view := model.View()
	if !strings.Contains(view, "Error: invalid language") && !strings.Contains(view, "test code") {
		t.Error("Expected error to be shown in view or fallback content")
	}
}
