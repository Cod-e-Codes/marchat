package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/alecthomas/chroma/quick"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type codeSnippetState int

const (
	stateSelectLang codeSnippetState = iota
	stateInputCode
	stateConfirmSend
)

// Wrap language strings into a type that implements list.Item
type langItem string

func (l langItem) Title() string       { return string(l) }
func (l langItem) Description() string { return "" }
func (l langItem) FilterValue() string { return string(l) }

type codeSnippetModel struct {
	state     codeSnippetState
	languages []string
	selected  string
	code      string
	lines     []string
	cursorX   int
	cursorY   int
	// Text selection
	selectionStartX int
	selectionStartY int
	selectionEndX   int
	selectionEndY   int
	hasSelection    bool
	langList        list.Model
	highlight       string
	styles          themeStyles
	width           int
	height          int
	// Integration with main model
	onSend   func(string) // Callback to send the highlighted code
	onCancel func()       // Callback to cancel
	// For sub-model integration
	initialized bool
}

func newCodeSnippetModel(styles themeStyles, width, height int, onSend func(string), onCancel func()) codeSnippetModel {
	languages := []string{
		"go", "python", "javascript", "typescript", "java", "c", "cpp", "csharp",
		"rust", "php", "ruby", "swift", "kotlin", "scala", "haskell", "clojure",
		"lua", "perl", "bash", "powershell", "sql", "html", "css", "json",
		"yaml", "xml", "markdown", "dockerfile", "makefile", "vim", "diff",
	}

	items := make([]list.Item, len(languages))
	for i, lang := range languages {
		items[i] = langItem(lang)
	}

	l := list.New(items, list.NewDefaultDelegate(), width-4, height-8)
	l.Title = "Select Programming Language"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	return codeSnippetModel{
		state:       stateSelectLang,
		languages:   languages,
		langList:    l,
		lines:       []string{""},
		cursorX:     0,
		cursorY:     0,
		styles:      styles,
		width:       width,
		height:      height,
		onSend:      onSend,
		onCancel:    onCancel,
		initialized: true,
	}
}

// Helper functions for text selection and clipboard operations
func (m *codeSnippetModel) clearSelection() {
	m.hasSelection = false
	m.selectionStartX = 0
	m.selectionStartY = 0
	m.selectionEndX = 0
	m.selectionEndY = 0
}

func (m *codeSnippetModel) startSelection() {
	m.selectionStartX = m.cursorX
	m.selectionStartY = m.cursorY
	m.selectionEndX = m.cursorX
	m.selectionEndY = m.cursorY
	m.hasSelection = true
}

func (m *codeSnippetModel) updateSelection() {
	if m.hasSelection {
		m.selectionEndX = m.cursorX
		m.selectionEndY = m.cursorY
	}
}

func (m *codeSnippetModel) getSelectedText() string {
	if !m.hasSelection {
		return ""
	}

	startX, startY, endX, endY := m.getSelectionBounds()

	if startY == endY {
		// Single line selection
		return m.lines[startY][startX:endX]
	}

	var result strings.Builder
	// First line
	result.WriteString(m.lines[startY][startX:])
	result.WriteString("\n")

	// Middle lines
	for i := startY + 1; i < endY; i++ {
		result.WriteString(m.lines[i])
		result.WriteString("\n")
	}

	// Last line
	result.WriteString(m.lines[endY][:endX])

	return result.String()
}

func (m *codeSnippetModel) getSelectionBounds() (startX, startY, endX, endY int) {
	if !m.hasSelection {
		return 0, 0, 0, 0
	}

	startX, startY = m.selectionStartX, m.selectionStartY
	endX, endY = m.selectionEndX, m.selectionEndY

	// Ensure start is before end
	if startY > endY || (startY == endY && startX > endX) {
		startX, startY, endX, endY = endX, endY, startX, startY
	}

	return startX, startY, endX, endY
}

func (m *codeSnippetModel) deleteSelection() {
	if !m.hasSelection {
		return
	}

	startX, startY, endX, endY := m.getSelectionBounds()

	if startY == endY {
		// Single line selection
		line := m.lines[startY]
		m.lines[startY] = line[:startX] + line[endX:]
		m.cursorX = startX
	} else {
		// Multi-line selection
		// Keep the part before selection on first line
		firstLine := m.lines[startY][:startX]
		// Keep the part after selection on last line
		lastLine := m.lines[endY][endX:]
		// Combine them
		m.lines[startY] = firstLine + lastLine

		// Remove the lines in between
		m.lines = append(m.lines[:startY+1], m.lines[endY+1:]...)
		m.cursorX = startX
		m.cursorY = startY
	}

	m.clearSelection()
}

func (m *codeSnippetModel) selectAll() {
	if len(m.lines) == 0 {
		return
	}

	m.selectionStartX = 0
	m.selectionStartY = 0
	m.selectionEndX = len(m.lines[len(m.lines)-1])
	m.selectionEndY = len(m.lines) - 1
	m.hasSelection = true
}

func (m *codeSnippetModel) pasteText(text string) {
	if m.hasSelection {
		m.deleteSelection()
	}

	// Split text into lines
	lines := strings.Split(text, "\n")

	if len(lines) == 1 {
		// Single line paste
		currentLine := m.lines[m.cursorY]
		m.lines[m.cursorY] = currentLine[:m.cursorX] + lines[0] + currentLine[m.cursorX:]
		m.cursorX += len(lines[0])
	} else {
		// Multi-line paste
		currentLine := m.lines[m.cursorY]
		beforeCursor := currentLine[:m.cursorX]
		afterCursor := currentLine[m.cursorX:]

		// Create new lines array
		newLines := make([]string, len(m.lines)+len(lines)-1)
		copy(newLines, m.lines[:m.cursorY])

		// First line of paste
		newLines[m.cursorY] = beforeCursor + lines[0]

		// Middle lines of paste
		for i := 1; i < len(lines)-1; i++ {
			newLines[m.cursorY+i] = lines[i]
		}

		// Last line of paste
		newLines[m.cursorY+len(lines)-1] = lines[len(lines)-1] + afterCursor

		// Remaining original lines
		copy(newLines[m.cursorY+len(lines):], m.lines[m.cursorY+1:])

		m.lines = newLines
		m.cursorX = len(lines[len(lines)-1])
		m.cursorY = m.cursorY + len(lines) - 1
	}
}

func (m codeSnippetModel) Init() tea.Cmd {
	return nil
}

func (m codeSnippetModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateSelectLang:
		var cmd tea.Cmd
		m.langList, cmd = m.langList.Update(msg)
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				if item, ok := m.langList.SelectedItem().(langItem); ok {
					m.selected = string(item)
				} else {
					m.selected = m.languages[m.langList.Index()]
				}
				m.state = stateInputCode
			case "esc", "ctrl+c":
				m.onCancel()
				return m, nil
			}
		}
		return m, cmd

	case stateInputCode:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+s":
				// Ctrl+S to finish input and show preview
				m.code = strings.Join(m.lines, "\n")
				var sb strings.Builder
				err := quick.Highlight(&sb, m.code, m.selected, "terminal256", "monokai")
				if err != nil {
					m.highlight = fmt.Sprintf("Error: %v", err)
				} else {
					m.highlight = sb.String()
				}
				m.state = stateConfirmSend
			case "ctrl+c":
				// Copy selected text to clipboard
				if m.hasSelection {
					selectedText := m.getSelectedText()
					if err := clipboard.WriteAll(selectedText); err != nil {
						log.Printf("Failed to copy to clipboard: %v", err)
					}
				}
				return m, nil
			case "ctrl+x":
				// Cut selected text to clipboard
				if m.hasSelection {
					selectedText := m.getSelectedText()
					if err := clipboard.WriteAll(selectedText); err != nil {
						log.Printf("Failed to copy to clipboard: %v", err)
					}
					m.deleteSelection()
				}
				return m, nil
			case "ctrl+v":
				// Paste from clipboard
				clipboardText, err := clipboard.ReadAll()
				if err != nil {
					log.Printf("Failed to read from clipboard: %v", err)
				} else {
					m.pasteText(clipboardText)
				}
				return m, nil
			case "ctrl+a":
				// Select all text
				m.selectAll()
				return m, nil
			case "esc":
				m.onCancel()
				return m, nil
			case "enter":
				// Regular Enter - add new line
				currentLine := m.lines[m.cursorY]
				beforeCursor := currentLine[:m.cursorX]
				afterCursor := currentLine[m.cursorX:]

				// Split current line
				m.lines[m.cursorY] = beforeCursor

				// Insert new line
				newLines := make([]string, len(m.lines)+1)
				copy(newLines, m.lines[:m.cursorY+1])
				newLines[m.cursorY+1] = afterCursor
				copy(newLines[m.cursorY+2:], m.lines[m.cursorY+1:])
				m.lines = newLines

				// Move cursor to start of new line
				m.cursorY++
				m.cursorX = 0
			case "backspace":
				if m.hasSelection {
					m.deleteSelection()
				} else if m.cursorX > 0 {
					// Delete character before cursor
					currentLine := m.lines[m.cursorY]
					m.lines[m.cursorY] = currentLine[:m.cursorX-1] + currentLine[m.cursorX:]
					m.cursorX--
				} else if m.cursorY > 0 {
					// Merge with previous line
					prevLine := m.lines[m.cursorY-1]
					currentLine := m.lines[m.cursorY]
					m.lines[m.cursorY-1] = prevLine + currentLine

					// Remove current line
					m.lines = append(m.lines[:m.cursorY], m.lines[m.cursorY+1:]...)
					m.cursorY--
					m.cursorX = len(prevLine)
				}
			case "delete":
				if m.hasSelection {
					m.deleteSelection()
				} else if m.cursorX < len(m.lines[m.cursorY]) {
					// Delete character at cursor
					currentLine := m.lines[m.cursorY]
					m.lines[m.cursorY] = currentLine[:m.cursorX] + currentLine[m.cursorX+1:]
				} else if m.cursorY < len(m.lines)-1 {
					// Merge with next line
					currentLine := m.lines[m.cursorY]
					nextLine := m.lines[m.cursorY+1]
					m.lines[m.cursorY] = currentLine + nextLine

					// Remove next line
					m.lines = append(m.lines[:m.cursorY+1], m.lines[m.cursorY+2:]...)
				}
			case "left":
				if m.cursorX > 0 {
					m.cursorX--
				} else if m.cursorY > 0 {
					m.cursorY--
					m.cursorX = len(m.lines[m.cursorY])
				}
				m.updateSelection()
			case "right":
				if m.cursorX < len(m.lines[m.cursorY]) {
					m.cursorX++
				} else if m.cursorY < len(m.lines)-1 {
					m.cursorY++
					m.cursorX = 0
				}
				m.updateSelection()
			case "up":
				if m.cursorY > 0 {
					m.cursorY--
					if m.cursorX > len(m.lines[m.cursorY]) {
						m.cursorX = len(m.lines[m.cursorY])
					}
				}
				m.updateSelection()
			case "down":
				if m.cursorY < len(m.lines)-1 {
					m.cursorY++
					if m.cursorX > len(m.lines[m.cursorY]) {
						m.cursorX = len(m.lines[m.cursorY])
					}
				}
				m.updateSelection()
			case "shift+left":
				if !m.hasSelection {
					m.startSelection()
				}
				if m.cursorX > 0 {
					m.cursorX--
				} else if m.cursorY > 0 {
					m.cursorY--
					m.cursorX = len(m.lines[m.cursorY])
				}
				m.updateSelection()
			case "shift+right":
				if !m.hasSelection {
					m.startSelection()
				}
				if m.cursorX < len(m.lines[m.cursorY]) {
					m.cursorX++
				} else if m.cursorY < len(m.lines)-1 {
					m.cursorY++
					m.cursorX = 0
				}
				m.updateSelection()
			case "shift+up":
				if !m.hasSelection {
					m.startSelection()
				}
				if m.cursorY > 0 {
					m.cursorY--
					if m.cursorX > len(m.lines[m.cursorY]) {
						m.cursorX = len(m.lines[m.cursorY])
					}
				}
				m.updateSelection()
			case "shift+down":
				if !m.hasSelection {
					m.startSelection()
				}
				if m.cursorY < len(m.lines)-1 {
					m.cursorY++
					if m.cursorX > len(m.lines[m.cursorY]) {
						m.cursorX = len(m.lines[m.cursorY])
					}
				}
				m.updateSelection()
			case "home":
				m.cursorX = 0
			case "end":
				m.cursorX = len(m.lines[m.cursorY])
			default:
				// Handle regular character input
				if len(msg.String()) == 1 {
					// Clear selection if any
					if m.hasSelection {
						m.deleteSelection()
					}

					char := msg.String()
					currentLine := m.lines[m.cursorY]
					m.lines[m.cursorY] = currentLine[:m.cursorX] + char + currentLine[m.cursorX:]
					m.cursorX++
				}
			}
		}
		return m, nil

	case stateConfirmSend:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				// Send the highlighted code as a message
				markdown := fmt.Sprintf("```%s\n%s\n```", m.selected, m.code)
				m.onSend(markdown)
				m.onCancel() // Close the interface after sending
				return m, nil
			case "esc", "ctrl+c":
				m.onCancel()
				return m, nil
			case "r":
				// restart input
				m.state = stateSelectLang
				m.lines = []string{""}
				m.cursorX = 0
				m.cursorY = 0
			case "c":
				// copy to clipboard - sanitize the code first
				sanitizedCode := strings.ReplaceAll(m.code, "\x00", "")
				if err := clipboard.WriteAll(sanitizedCode); err != nil {
					// Silently ignore clipboard errors - not critical for functionality
					log.Printf("Failed to copy to clipboard: %v", err)
				}
			}
		}
		return m, nil
	}

	return m, nil
}

func (m codeSnippetModel) View() string {
	switch m.state {
	case stateSelectLang:
		return m.langList.View() + "\n" + m.styles.Time.Render("Press Enter to select language, Esc to cancel.")
	case stateInputCode:
		var sb strings.Builder
		sb.WriteString(m.styles.User.Render(fmt.Sprintf("Language: %s", m.selected)) + "\n\n")

		// Display all lines with cursor and selection
		for i, line := range m.lines {
			if i == m.cursorY {
				// Current line with cursor
				beforeCursor := line[:m.cursorX]
				afterCursor := line[m.cursorX:]

				// Check if this line has selection
				if m.hasSelection {
					startX, startY, endX, endY := m.getSelectionBounds()
					if i >= startY && i <= endY {
						// This line is part of the selection
						lineStart := 0
						lineEnd := len(line)

						if i == startY {
							lineStart = startX
						}
						if i == endY {
							lineEnd = endX
						}

						// Render with selection highlighting
						beforeSelection := line[:lineStart]
						selection := line[lineStart:lineEnd]
						afterSelection := line[lineEnd:]

						// Use different style for selected text
						selectedStyle := m.styles.Banner // Use banner style for selection
						sb.WriteString(m.styles.Msg.Render(fmt.Sprintf("> %s", beforeSelection)))
						sb.WriteString(selectedStyle.Render(selection))
						sb.WriteString(m.styles.Msg.Render(fmt.Sprintf("%s|%s", afterSelection, afterCursor)) + "\n")
					} else {
						sb.WriteString(m.styles.Msg.Render(fmt.Sprintf("> %s|%s", beforeCursor, afterCursor)) + "\n")
					}
				} else {
					sb.WriteString(m.styles.Msg.Render(fmt.Sprintf("> %s|%s", beforeCursor, afterCursor)) + "\n")
				}
			} else {
				// Other lines - check if they have selection
				if m.hasSelection {
					startX, startY, endX, endY := m.getSelectionBounds()
					if i >= startY && i <= endY {
						// This line is part of the selection
						lineStart := 0
						lineEnd := len(line)

						if i == startY {
							lineStart = startX
						}
						if i == endY {
							lineEnd = endX
						}

						// Render with selection highlighting
						beforeSelection := line[:lineStart]
						selection := line[lineStart:lineEnd]
						afterSelection := line[lineEnd:]

						// Use different style for selected text
						selectedStyle := m.styles.Banner // Use banner style for selection
						sb.WriteString(m.styles.Msg.Render(fmt.Sprintf("  %s", beforeSelection)))
						sb.WriteString(selectedStyle.Render(selection))
						sb.WriteString(m.styles.Msg.Render(afterSelection) + "\n")
					} else {
						sb.WriteString(m.styles.Msg.Render(fmt.Sprintf("  %s", line)) + "\n")
					}
				} else {
					sb.WriteString(m.styles.Msg.Render(fmt.Sprintf("  %s", line)) + "\n")
				}
			}
		}

		sb.WriteString("\n" + m.styles.Time.Render("Press Ctrl+S to preview, Ctrl+C/V/X/A for copy/paste/cut/select all, Esc to cancel."))
		return sb.String()
	case stateConfirmSend:
		return m.formatCodeBlock(m.selected, m.code, false)
	default:
		return "Unknown state"
	}
}

func (m codeSnippetModel) formatCodeBlock(language, plainCode string, copied bool) string {
	// Use Chroma directly for syntax highlighting
	var sb strings.Builder
	err := quick.Highlight(&sb, plainCode, language, "terminal256", "monokai")
	if err != nil {
		// Fallback to simple formatting if highlighting fails
		return fmt.Sprintf("```%s\n%s\n```\n\n%s", language, plainCode, m.styles.Time.Render("Press Enter to send, 'r' to restart, Esc to cancel."))
	}

	highlighted := sb.String()

	// Add status message
	if copied {
		highlighted += "\n" + m.styles.Banner.Render("[OK] Code copied to clipboard!")
	} else {
		highlighted += "\n" + m.styles.Time.Render("Press Enter to send message, 'r' to restart, 'c' to copy, Esc to cancel.")
	}

	return highlighted
}
