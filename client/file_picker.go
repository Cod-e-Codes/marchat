package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// File picker states
const (
	stateSelectFile = iota
	stateConfirmFileSend
)

// File item for the list
type fileItem struct {
	title       string
	description string
	path        string
	isDir       bool
}

func (f fileItem) Title() string       { return f.title }
func (f fileItem) Description() string { return f.description }
func (f fileItem) FilterValue() string { return f.title }

// File picker model
type filePickerModel struct {
	state        int
	list         list.Model
	selectedFile string
	fileSize     int64
	err          error
	styles       themeStyles
	width        int
	height       int
	onSend       func(string)
	onCancel     func()
	initialized  bool
}

// Error message types
type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func newFilePickerModel(styles themeStyles, width, height int, onSend func(string), onCancel func()) filePickerModel {
	// Get current directory
	currentDir := "."
	if cwd, err := os.Getwd(); err == nil {
		currentDir = cwd
	}

	// Create list items from directory contents
	items := createFileListItems(currentDir)

	// Create list
	l := list.New(items, list.NewDefaultDelegate(), width-4, height-8)
	l.Title = "Select File to Send"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFDF5")).
		Background(lipgloss.Color("#25A065")).
		Padding(0, 1)

	return filePickerModel{
		state:       stateSelectFile,
		list:        l,
		styles:      styles,
		width:       width,
		height:      height,
		onSend:      onSend,
		onCancel:    onCancel,
		initialized: true,
	}
}

func createFileListItems(dir string) []list.Item {
	var items []list.Item

	// Add parent directory option if not at root
	if dir != "/" && dir != "." {
		parentDir := filepath.Dir(dir)
		items = append(items, fileItem{
			title:       ".. (Parent Directory)",
			description: "Go up one level",
			path:        parentDir,
			isDir:       true,
		})
	}

	// Read directory contents
	files, err := os.ReadDir(dir)
	if err != nil {
		return items
	}

	// Add directories first
	for _, file := range files {
		if file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
			items = append(items, fileItem{
				title:       "[DIR] " + file.Name(),
				description: "Directory",
				path:        filepath.Join(dir, file.Name()),
				isDir:       true,
			})
		}
	}

	// Add files
	for _, file := range files {
		if !file.IsDir() && isAllowedFileType(file.Name()) {
			info, err := file.Info()
			size := int64(0)
			if err == nil {
				size = info.Size()
			}

			items = append(items, fileItem{
				title:       "[FILE] " + file.Name(),
				description: formatFileSize(size),
				path:        filepath.Join(dir, file.Name()),
				isDir:       false,
			})
		}
	}

	return items
}

func isAllowedFileType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	allowedTypes := []string{
		".txt", ".md", ".json", ".yaml", ".yml", ".xml", ".csv",
		".log", ".conf", ".config", ".ini", ".cfg",
		".go", ".py", ".js", ".ts", ".java", ".c", ".cpp", ".h",
		".html", ".css", ".scss", ".less",
		".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".zip", ".tar", ".gz", ".rar",
	}

	for _, allowed := range allowedTypes {
		if ext == allowed {
			return true
		}
	}
	return false
}

func (m filePickerModel) Init() tea.Cmd {
	return nil
}

func (m filePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case clearErrorMsg:
		m.err = nil
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case stateSelectFile:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				// Get selected item
				if selectedItem, ok := m.list.SelectedItem().(fileItem); ok {
					if selectedItem.isDir {
						// Navigate to directory
						items := createFileListItems(selectedItem.path)
						m.list.SetItems(items)
						m.list.Select(0) // Select first item
					} else {
						// Check file size (configurable; default 1MB)
						if info, err := os.Stat(selectedItem.path); err == nil {
							var maxBytes int64 = 1024 * 1024
							if envBytes := os.Getenv("MARCHAT_MAX_FILE_BYTES"); envBytes != "" {
								if v, err := strconv.ParseInt(envBytes, 10, 64); err == nil && v > 0 {
									maxBytes = v
								}
							} else if envMB := os.Getenv("MARCHAT_MAX_FILE_MB"); envMB != "" {
								if v, err := strconv.ParseInt(envMB, 10, 64); err == nil && v > 0 {
									maxBytes = v * 1024 * 1024
								}
							}
							if info.Size() > maxBytes {
								// Build friendly limit message
								limitMsg := fmt.Sprintf("%d bytes", maxBytes)
								if maxBytes%(1024*1024) == 0 {
									limitMsg = fmt.Sprintf("%dMB", maxBytes/(1024*1024))
								}
								m.err = fmt.Errorf("file too large (max %s)", limitMsg)
								return m, clearErrorAfter(2 * time.Second)
							}
							m.selectedFile = selectedItem.path
							m.fileSize = info.Size()
							m.state = stateConfirmFileSend
						} else {
							m.err = fmt.Errorf("failed to read file info")
							return m, clearErrorAfter(2 * time.Second)
						}
					}
				}
				return m, nil

			case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "ctrl+c"))):
				m.onCancel()
				return m, nil
			}

		case stateConfirmFileSend:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				// Send the file
				m.onSend(m.selectedFile)
				m.onCancel() // Close the interface after sending
				return m, nil

			case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "ctrl+c"))):
				m.onCancel()
				return m, nil

			case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
				// Go back to file selection
				m.state = stateSelectFile
				m.selectedFile = ""
				m.fileSize = 0
			}
		}

		// Update the list
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m filePickerModel) View() string {
	switch m.state {
	case stateSelectFile:
		var s strings.Builder
		s.WriteString(m.styles.HelpTitle.Render("Select File to Send") + "\n\n")

		if m.err != nil {
			s.WriteString(m.styles.Banner.Render("[ERROR] "+m.err.Error()) + "\n\n")
		} else {
			s.WriteString(m.styles.Time.Render("Navigate with arrow keys, Enter to select, Esc to cancel") + "\n\n")
		}

		s.WriteString(m.list.View())
		return s.String()

	case stateConfirmFileSend:
		var s strings.Builder
		filename := filepath.Base(m.selectedFile)
		fileSizeStr := formatFileSize(m.fileSize)

		s.WriteString(m.styles.HelpTitle.Render("Confirm File Send") + "\n\n")
		s.WriteString(m.styles.User.Render("File: ") + m.styles.Msg.Render(filename) + "\n")
		s.WriteString(m.styles.User.Render("Path: ") + m.styles.Msg.Render(m.selectedFile) + "\n")
		s.WriteString(m.styles.User.Render("Size: ") + m.styles.Msg.Render(fileSizeStr) + "\n\n")
		s.WriteString(m.styles.Time.Render("Press Enter to send, 'r' to select different file, Esc to cancel"))

		return s.String()

	default:
		return "Unknown state"
	}
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
