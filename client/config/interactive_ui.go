package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B9D"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	cursorStyle  = focusedStyle
	noStyle      = lipgloss.NewStyle()
	helpStyle    = blurredStyle
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFEAA7")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#4CAF50"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00BCD4"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))

	focusedButton = focusedStyle.Render("[ Connect ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Connect"))
)

type configField int

const (
	serverURLField configField = iota
	usernameField
	adminField
	adminKeyField
	e2eField
	keystorePassField
	themeField
	submitButton
)

type ConfigUIModel struct {
	focusIndex    int
	inputs        []textinput.Model
	config        *Config
	errorMessage  string
	showAdminKey  bool
	showE2EFields bool
	finished      bool
	cancelled     bool
}

func NewConfigUI() ConfigUIModel {
	m := ConfigUIModel{
		inputs: make([]textinput.Model, 7), // 7 input fields
		config: &Config{
			Theme:          "system",
			TwentyFourHour: true,
		},
	}

	// Initialize text inputs
	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle

		switch configField(i) {
		case serverURLField:
			t.Placeholder = "wss://example.com/ws"
			t.Prompt = "Server URL: "
			t.CharLimit = 256
			t.Width = 50
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case usernameField:
			t.Placeholder = "Enter your username"
			t.Prompt = "Username: "
			t.CharLimit = 32
			t.Width = 30
		case adminField:
			t.Placeholder = "y/n"
			t.Prompt = "Admin user? "
			t.CharLimit = 1
			t.Width = 5
		case adminKeyField:
			t.Placeholder = "Enter admin key"
			t.Prompt = "Admin Key: "
			t.CharLimit = 64
			t.Width = 40
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		case e2eField:
			t.Placeholder = "y/n"
			t.Prompt = "Enable E2E encryption? "
			t.CharLimit = 1
			t.Width = 5
		case keystorePassField:
			t.Placeholder = "Enter keystore passphrase"
			t.Prompt = "Keystore passphrase: "
			t.CharLimit = 128
			t.Width = 40
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		case themeField:
			t.Placeholder = "system"
			t.Prompt = "Theme: "
			t.CharLimit = 20
			t.Width = 20
		}

		m.inputs[i] = t
	}

	return m
}

func (m ConfigUIModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ConfigUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Handle submit
			if s == "enter" && m.focusIndex == int(submitButton) {
				if err := m.validateAndBuildConfig(); err != nil {
					m.errorMessage = err.Error()
					return m, nil
				}
				m.finished = true
				return m, tea.Quit
			}

			// Update conditional field visibility on any navigation
			m.updateConditionalFields()

			// Navigate between fields
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			// Skip conditional fields that shouldn't be shown
			m.focusIndex = m.getNextValidFocus(m.focusIndex, s == "up" || s == "shift+tab")

			if m.focusIndex > int(submitButton) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = int(submitButton)
			}

			m.updateFocus()
			return m, nil
		}
	}

	// Handle character input
	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *ConfigUIModel) updateConditionalFields() {
	// Update admin key field visibility
	adminValue := strings.ToLower(strings.TrimSpace(m.inputs[adminField].Value()))
	m.showAdminKey = adminValue == "y" || adminValue == "yes"
	if !m.showAdminKey {
		m.inputs[adminKeyField].SetValue("")
	}

	// Update E2E field visibility
	e2eValue := strings.ToLower(strings.TrimSpace(m.inputs[e2eField].Value()))
	m.showE2EFields = e2eValue == "y" || e2eValue == "yes"
	if !m.showE2EFields {
		m.inputs[keystorePassField].SetValue("")
	}
}

func (m *ConfigUIModel) getNextValidFocus(index int, reverse bool) int {
	for {
		if index < 0 {
			return int(submitButton)
		}
		if index > int(submitButton) {
			return 0
		}

		field := configField(index)

		// Skip admin key field if not admin
		if field == adminKeyField && !m.showAdminKey {
			if reverse {
				index--
			} else {
				index++
			}
			continue
		}

		// Skip keystore passphrase field if E2E not enabled
		if field == keystorePassField && !m.showE2EFields {
			if reverse {
				index--
			} else {
				index++
			}
			continue
		}

		break
	}
	return index
}

func (m *ConfigUIModel) updateFocus() {
	for i := 0; i < len(m.inputs); i++ {
		if i == m.focusIndex {
			m.inputs[i].Focus()
			m.inputs[i].PromptStyle = focusedStyle
			m.inputs[i].TextStyle = focusedStyle
		} else {
			m.inputs[i].Blur()
			m.inputs[i].PromptStyle = noStyle
			m.inputs[i].TextStyle = noStyle
		}
	}
}

func (m *ConfigUIModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m *ConfigUIModel) validateAndBuildConfig() error {
	// Clear previous error
	m.errorMessage = ""

	// Get values
	serverURL := strings.TrimSpace(m.inputs[serverURLField].Value())
	username := strings.TrimSpace(m.inputs[usernameField].Value())
	adminStr := strings.ToLower(strings.TrimSpace(m.inputs[adminField].Value()))
	adminKey := strings.TrimSpace(m.inputs[adminKeyField].Value())
	e2eStr := strings.ToLower(strings.TrimSpace(m.inputs[e2eField].Value()))
	keystorePass := m.inputs[keystorePassField].Value()
	theme := strings.TrimSpace(m.inputs[themeField].Value())

	// Validation
	if serverURL == "" {
		return fmt.Errorf("server URL is required")
	}
	if username == "" {
		return fmt.Errorf("username is required")
	}

	isAdmin := adminStr == "y" || adminStr == "yes"
	useE2E := e2eStr == "y" || e2eStr == "yes"

	if isAdmin && adminKey == "" {
		return fmt.Errorf("admin key is required for admin users")
	}

	if useE2E && keystorePass == "" {
		return fmt.Errorf("keystore passphrase is required for E2E encryption")
	}

	if theme == "" {
		theme = "system"
	}

	// Build config
	m.config = &Config{
		Username:       username,
		ServerURL:      serverURL,
		IsAdmin:        isAdmin,
		AdminKey:       adminKey,
		UseE2E:         useE2E,
		Theme:          theme,
		TwentyFourHour: true,
	}

	return nil
}

func (m ConfigUIModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("marchat Configuration"))
	b.WriteString("\n\n")

	// Server URL
	b.WriteString(m.inputs[serverURLField].View())
	b.WriteString("\n")

	// Username
	b.WriteString(m.inputs[usernameField].View())
	b.WriteString("\n")

	// Admin
	b.WriteString(m.inputs[adminField].View())
	if m.focusIndex == int(adminField) {
		b.WriteString(helpStyle.Render(" (y/n)"))
	}
	b.WriteString("\n")

	// Admin key (conditional)
	if m.showAdminKey {
		b.WriteString(m.inputs[adminKeyField].View())
		b.WriteString("\n")
	}

	// E2E Encryption
	b.WriteString(m.inputs[e2eField].View())
	if m.focusIndex == int(e2eField) {
		b.WriteString(helpStyle.Render(" (y/n)"))
	}
	b.WriteString("\n")

	// Keystore passphrase (conditional)
	if m.showE2EFields {
		b.WriteString(m.inputs[keystorePassField].View())
		b.WriteString("\n")
	}

	// Theme
	b.WriteString(m.inputs[themeField].View())
	if m.focusIndex == int(themeField) {
		b.WriteString(helpStyle.Render(" (system, patriot, retro, modern, or custom)"))
	}
	b.WriteString("\n\n")

	// Submit button
	button := &blurredButton
	if m.focusIndex == int(submitButton) {
		button = &focusedButton
	}
	b.WriteString(*button)
	b.WriteString("\n\n")

	// Error message
	if m.errorMessage != "" {
		b.WriteString(errorStyle.Render("ERROR: " + m.errorMessage))
		b.WriteString("\n\n")
	}

	// Help
	b.WriteString(helpStyle.Render("Tab/Shift+Tab: Navigate • Enter: Select/Submit • Esc: Cancel"))

	return b.String()
}

// GetConfig returns the built configuration
func (m ConfigUIModel) GetConfig() *Config {
	return m.config
}

// IsFinished returns true if the user completed the configuration
func (m ConfigUIModel) IsFinished() bool {
	return m.finished
}

// IsCancelled returns true if the user cancelled the configuration
func (m ConfigUIModel) IsCancelled() bool {
	return m.cancelled
}

// GetKeystorePassphrase returns the keystore passphrase
func (m ConfigUIModel) GetKeystorePassphrase() string {
	if !m.showE2EFields {
		return ""
	}
	return m.inputs[keystorePassField].Value()
}

// ProfileOperation represents different operations that can be performed
type ProfileOperation int

const (
	ProfileOpNone ProfileOperation = iota
	ProfileOpView
	ProfileOpRename
	ProfileOpDelete
)

// ProfileSelectionModel handles profile selection UI with management features
type ProfileSelectionModel struct {
	profiles        []ConnectionProfile
	cursor          int
	selected        bool
	cancelled       bool
	choice          int
	showNewOption   bool // Whether to show "Create New Profile" option
	operation       ProfileOperation
	renameInput     textinput.Model
	message         string
	messageType     string                   // "success", "error", "warning", "info"
	deleteConfirm   string                   // stores the profile name being deleted
	modified        bool                     // tracks if profiles were modified
	icl             *InteractiveConfigLoader // for saving profiles
	selectedProfile *ConnectionProfile       // Store the actual selected profile
}

func NewProfileSelectionModel(profiles []ConnectionProfile, showNewOption bool) ProfileSelectionModel {
	return ProfileSelectionModel{
		profiles:      profiles,
		cursor:        0,
		showNewOption: showNewOption,
		operation:     ProfileOpNone,
	}
}

func NewEnhancedProfileSelectionModel(profiles []ConnectionProfile, showNewOption bool, icl *InteractiveConfigLoader) ProfileSelectionModel {
	// Initialize rename input
	ti := textinput.New()
	ti.Cursor.Style = cursorStyle
	ti.CharLimit = 50
	ti.Width = 40
	ti.Prompt = "New name: "
	ti.PromptStyle = focusedStyle
	ti.TextStyle = focusedStyle

	return ProfileSelectionModel{
		profiles:      profiles,
		cursor:        0,
		showNewOption: showNewOption,
		operation:     ProfileOpNone,
		renameInput:   ti,
		icl:           icl,
	}
}

func (m ProfileSelectionModel) Init() tea.Cmd {
	return nil
}

func (m ProfileSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle operations first
	switch m.operation {
	case ProfileOpView:
		return m.handleViewOperation(msg)
	case ProfileOpRename:
		return m.handleRenameOperation(msg)
	case ProfileOpDelete:
		return m.handleDeleteOperation(msg)
	}

	// Handle main selection
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Clear message on any key press if it's been shown
		if m.message != "" && m.messageType != "error" {
			m.message = ""
			m.messageType = ""
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			maxCursor := len(m.profiles) - 1
			if m.showNewOption {
				maxCursor++
			}
			if m.cursor < maxCursor {
				m.cursor++
			}
		case "enter":
			if m.showNewOption && m.cursor == len(m.profiles) {
				// Selected "Create New Profile"
				m.choice = m.cursor
				m.selected = true
				// selectedProfile remains nil for "create new"
				return m, tea.Quit
			} else if m.cursor < len(m.profiles) {
				m.choice = m.cursor
				m.selected = true
				// CRITICAL FIX: Store the actual profile, not just the index
				profile := m.profiles[m.cursor]
				m.selectedProfile = &profile
				return m, tea.Quit
			}
		case "i", "v": // View details
			if m.cursor < len(m.profiles) {
				m.operation = ProfileOpView
			}
		case "r": // Rename
			if m.cursor < len(m.profiles) {
				m.operation = ProfileOpRename
				m.renameInput.SetValue(m.profiles[m.cursor].Name)
				m.renameInput.Focus()
				return m, textinput.Blink
			}
		case "d": // Delete
			if m.cursor < len(m.profiles) {
				if len(m.profiles) == 1 {
					m.message = "Cannot delete the only profile"
					m.messageType = "error"
				} else {
					m.operation = ProfileOpDelete
					m.deleteConfirm = m.profiles[m.cursor].Name
				}
			}
		}
	}
	return m, nil
}

func (m ProfileSelectionModel) handleViewOperation(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "i", "v":
			m.operation = ProfileOpNone
		}
	}
	return m, nil
}

func (m ProfileSelectionModel) handleRenameOperation(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.operation = ProfileOpNone
			m.renameInput.Blur()
			return m, nil
		case "enter":
			newName := strings.TrimSpace(m.renameInput.Value())
			if newName == "" {
				m.message = "Profile name cannot be empty"
				m.messageType = "error"
				return m, nil
			}

			// Check for duplicate names
			for i, p := range m.profiles {
				if i != m.cursor && p.Name == newName {
					m.message = fmt.Sprintf("Profile '%s' already exists", newName)
					m.messageType = "error"
					return m, nil
				}
			}

			// Rename the profile
			oldName := m.profiles[m.cursor].Name
			m.profiles[m.cursor].Name = newName

			// Save profiles
			if m.icl != nil {
				profiles := &Profiles{Profiles: m.profiles}
				if err := m.icl.SaveProfiles(profiles); err != nil {
					m.message = fmt.Sprintf("Failed to save: %v", err)
					m.messageType = "error"
					// Revert the change
					m.profiles[m.cursor].Name = oldName
				} else {
					m.message = fmt.Sprintf("Renamed '%s' to '%s'", oldName, newName)
					m.messageType = "success"
					m.modified = true
				}
			}

			m.operation = ProfileOpNone
			m.renameInput.Blur()
			return m, nil
		default:
			var cmd tea.Cmd
			m.renameInput, cmd = m.renameInput.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m ProfileSelectionModel) handleDeleteOperation(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "n", "N":
			m.operation = ProfileOpNone
			m.deleteConfirm = ""
		case "y", "Y":
			// Perform the deletion immediately
			deletedName := m.profiles[m.cursor].Name
			m.profiles = append(m.profiles[:m.cursor], m.profiles[m.cursor+1:]...)

			// Adjust cursor if necessary
			if m.cursor >= len(m.profiles) && m.cursor > 0 {
				m.cursor--
			}

			// Save profiles
			if m.icl != nil {
				profiles := &Profiles{Profiles: m.profiles}
				if err := m.icl.SaveProfiles(profiles); err != nil {
					m.message = fmt.Sprintf("Failed to delete: %v", err)
					m.messageType = "error"
				} else {
					m.message = fmt.Sprintf("Deleted profile '%s'", deletedName)
					m.messageType = "success"
					m.modified = true
				}
			}

			m.operation = ProfileOpNone
			m.deleteConfirm = ""
		}
	}
	return m, nil
}

func (m ProfileSelectionModel) View() string {
	// Show operation-specific views
	switch m.operation {
	case ProfileOpView:
		return m.viewDetails()
	case ProfileOpRename:
		return m.viewRename()
	case ProfileOpDelete:
		return m.viewDelete()
	}

	// Main profile selection view
	var b strings.Builder

	if m.showNewOption {
		b.WriteString(titleStyle.Render("Select a connection profile:"))
	} else {
		b.WriteString(titleStyle.Render("Quick Start - Select a connection:"))
	}
	b.WriteString("\n\n")

	// Show message if any
	if m.message != "" {
		switch m.messageType {
		case "success":
			b.WriteString(successStyle.Render("[OK] " + m.message))
		case "error":
			b.WriteString(errorStyle.Render("ERROR: " + m.message))
		case "warning":
			b.WriteString(warningStyle.Render("[WARN] " + m.message))
		case "info":
			b.WriteString(infoStyle.Render("[INFO] " + m.message))
		default:
			b.WriteString(m.message)
		}
		b.WriteString("\n\n")
	}

	for i, profile := range m.profiles {
		status := ""
		if profile.IsAdmin {
			status += " [Admin]"
		}
		if profile.UseE2E {
			status += " [E2E]"
		}
		if i == 0 && profile.LastUsed > 0 {
			status += " [Recent]"
		}

		line := fmt.Sprintf("%s (%s@%s)%s", profile.Name, profile.Username, profile.ServerURL, status)

		if i == m.cursor {
			b.WriteString(focusedStyle.Render("> " + line))
		} else {
			b.WriteString("  " + line)
		}
		b.WriteString("\n")
	}

	// Add "Create New Profile" option if enabled
	if m.showNewOption {
		newProfileLine := "Create New Profile"
		if m.cursor == len(m.profiles) {
			b.WriteString(focusedStyle.Render("> " + newProfileLine))
		} else {
			b.WriteString("  " + newProfileLine)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓: Navigate • Enter: Select • i: View • r: Rename • d: Delete • Esc: Cancel"))

	return b.String()
}

func (m ProfileSelectionModel) viewDetails() string {
	var b strings.Builder

	if m.cursor >= len(m.profiles) {
		return "No profile selected"
	}

	profile := m.profiles[m.cursor]

	b.WriteString(titleStyle.Render("Profile Details"))
	b.WriteString("\n\n")

	b.WriteString(focusedStyle.Render("Name: "))
	b.WriteString(profile.Name)
	b.WriteString("\n")

	b.WriteString(focusedStyle.Render("Server: "))
	b.WriteString(profile.ServerURL)
	b.WriteString("\n")

	b.WriteString(focusedStyle.Render("Username: "))
	b.WriteString(profile.Username)
	b.WriteString("\n")

	b.WriteString(focusedStyle.Render("Admin Access: "))
	if profile.IsAdmin {
		b.WriteString(successStyle.Render("Yes"))
	} else {
		b.WriteString(dimStyle.Render("No"))
	}
	b.WriteString("\n")

	b.WriteString(focusedStyle.Render("E2E Encryption: "))
	if profile.UseE2E {
		b.WriteString(successStyle.Render("Enabled"))
	} else {
		b.WriteString(dimStyle.Render("Disabled"))
	}
	b.WriteString("\n")

	if profile.Theme != "" {
		b.WriteString(focusedStyle.Render("Theme: "))
		b.WriteString(profile.Theme)
		b.WriteString("\n")
	}

	if profile.LastUsed > 0 {
		b.WriteString(focusedStyle.Render("Last Used: "))
		lastUsed := time.Unix(profile.LastUsed, 0)
		b.WriteString(lastUsed.Format("Jan 2, 2006 at 3:04 PM"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press i/v/q/Esc to return"))

	return b.String()
}

func (m ProfileSelectionModel) viewRename() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Rename Profile"))
	b.WriteString("\n\n")

	b.WriteString("Current name: ")
	b.WriteString(blurredStyle.Render(m.profiles[m.cursor].Name))
	b.WriteString("\n\n")

	b.WriteString(m.renameInput.View())
	b.WriteString("\n\n")

	if m.message != "" && m.messageType == "error" {
		b.WriteString(errorStyle.Render("ERROR: " + m.message))
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render("Enter: Save • Esc: Cancel"))

	return b.String()
}

func (m ProfileSelectionModel) viewDelete() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Delete Profile"))
	b.WriteString("\n\n")

	b.WriteString(warningStyle.Render("[WARN] Warning: This action cannot be undone!"))
	b.WriteString("\n\n")

	b.WriteString("Delete profile '")
	b.WriteString(focusedStyle.Render(m.deleteConfirm))
	b.WriteString("'?\n\n")

	profile := m.profiles[m.cursor]
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Server: %s\n", profile.ServerURL)))
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Username: %s\n", profile.Username)))

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("y: Confirm Delete • n/Esc: Cancel"))

	return b.String()
}

func (m ProfileSelectionModel) IsSelected() bool {
	return m.selected
}

func (m ProfileSelectionModel) IsCancelled() bool {
	return m.cancelled
}

func (m ProfileSelectionModel) GetChoice() int {
	return m.choice
}

func (m ProfileSelectionModel) IsModified() bool {
	return m.modified
}

// IsCreateNew returns true if user selected "Create New Profile"
func (m ProfileSelectionModel) IsCreateNew() bool {
	return m.showNewOption && m.choice == len(m.profiles)
}

// GetSelectedProfile returns the actual selected profile object
func (m ProfileSelectionModel) GetSelectedProfile() *ConnectionProfile {
	return m.selectedProfile
}

// RunProfileSelection runs the profile selection UI (for quick-start)
func RunProfileSelection(profiles []ConnectionProfile) (int, error) {
	model := NewProfileSelectionModel(profiles, false)

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return -1, fmt.Errorf("failed to run profile selection UI: %w", err)
	}

	selectionModel := finalModel.(ProfileSelectionModel)

	if selectionModel.IsCancelled() {
		return -1, fmt.Errorf("profile selection cancelled by user")
	}

	if !selectionModel.IsSelected() {
		return -1, fmt.Errorf("no profile selected")
	}

	return selectionModel.GetChoice(), nil
}

// RunProfileSelectionWithNew runs the profile selection UI with "Create New" option
// Deprecated: Use RunEnhancedProfileSelectionWithNew instead
func RunProfileSelectionWithNew(profiles []ConnectionProfile, icl *InteractiveConfigLoader) (*ConnectionProfile, bool, error) {
	// Use the enhanced version with management features
	return RunEnhancedProfileSelectionWithNew(profiles, icl)
}

// RunEnhancedProfileSelection runs the enhanced profile selection UI with management features
func RunEnhancedProfileSelection(profiles []ConnectionProfile, icl *InteractiveConfigLoader) (*ConnectionProfile, error) {
	model := NewEnhancedProfileSelectionModel(profiles, false, icl)

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run profile selection UI: %w", err)
	}

	selectionModel := finalModel.(ProfileSelectionModel)

	if selectionModel.IsCancelled() {
		return nil, fmt.Errorf("profile selection cancelled by user")
	}

	if !selectionModel.IsSelected() {
		return nil, fmt.Errorf("no profile selected")
	}

	return selectionModel.GetSelectedProfile(), nil
}

// RunEnhancedProfileSelectionWithNew runs the enhanced profile selection UI with "Create New" option
func RunEnhancedProfileSelectionWithNew(profiles []ConnectionProfile, icl *InteractiveConfigLoader) (*ConnectionProfile, bool, error) {
	model := NewEnhancedProfileSelectionModel(profiles, true, icl)

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return nil, false, fmt.Errorf("failed to run profile selection UI: %w", err)
	}

	selectionModel := finalModel.(ProfileSelectionModel)

	if selectionModel.IsCancelled() {
		return nil, false, fmt.Errorf("profile selection cancelled by user")
	}

	if !selectionModel.IsSelected() {
		return nil, false, fmt.Errorf("no profile selected")
	}

	isCreateNew := selectionModel.IsCreateNew()
	return selectionModel.GetSelectedProfile(), isCreateNew, nil
}

// SensitiveDataModel handles prompting for admin key and keystore passphrase
type SensitiveDataModel struct {
	focusIndex   int
	inputs       []textinput.Model
	isAdmin      bool
	useE2E       bool
	errorMessage string
	finished     bool
	cancelled    bool
	adminKey     string
	keystorePass string
}

func NewSensitiveDataPrompt(isAdmin, useE2E bool) SensitiveDataModel {
	inputCount := 0
	if isAdmin {
		inputCount++
	}
	if useE2E {
		inputCount++
	}

	m := SensitiveDataModel{
		inputs:  make([]textinput.Model, inputCount),
		isAdmin: isAdmin,
		useE2E:  useE2E,
	}

	idx := 0
	if isAdmin {
		t := textinput.New()
		t.Placeholder = "Enter admin key"
		t.Prompt = "Admin Key: "
		t.CharLimit = 64
		t.Width = 40
		t.EchoMode = textinput.EchoPassword
		t.EchoCharacter = '•'
		t.Focus()
		t.PromptStyle = focusedStyle
		t.TextStyle = focusedStyle
		t.Cursor.Style = cursorStyle
		m.inputs[idx] = t
		idx++
	}

	if useE2E {
		t := textinput.New()
		t.Placeholder = "Enter keystore passphrase"
		t.Prompt = "Keystore passphrase: "
		t.CharLimit = 128
		t.Width = 40
		t.EchoMode = textinput.EchoPassword
		t.EchoCharacter = '•'
		t.Cursor.Style = cursorStyle
		if !isAdmin {
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		}
		m.inputs[idx] = t
	}

	return m
}

func (m SensitiveDataModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SensitiveDataModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			// If there are multiple inputs and we're not on the last one, move to next
			if len(m.inputs) > 1 && m.focusIndex < len(m.inputs)-1 {
				m.focusIndex++
				m.updateFocus()
				return m, nil
			}

			// Validate and submit
			m.errorMessage = "" // Clear any previous errors

			if m.isAdmin {
				adminKey := strings.TrimSpace(m.inputs[0].Value())
				if adminKey == "" {
					m.errorMessage = "Admin key is required"
					return m, nil
				}
				m.adminKey = adminKey
			}

			if m.useE2E {
				idx := 0
				if m.isAdmin {
					idx = 1
				}
				keystorePass := strings.TrimSpace(m.inputs[idx].Value())
				if keystorePass == "" {
					m.errorMessage = "Keystore passphrase is required"
					return m, nil
				}
				m.keystorePass = keystorePass
			}

			m.finished = true
			return m, tea.Quit

		case "tab", "shift+tab", "up", "down":
			if len(m.inputs) > 1 {
				s := msg.String()
				if s == "up" || s == "shift+tab" {
					m.focusIndex--
				} else {
					m.focusIndex++
				}

				if m.focusIndex >= len(m.inputs) {
					m.focusIndex = 0
				} else if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs) - 1
				}

				m.updateFocus()
				return m, nil
			}
		}
	}

	// Handle character input
	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *SensitiveDataModel) updateFocus() {
	for i := 0; i < len(m.inputs); i++ {
		if i == m.focusIndex {
			m.inputs[i].Focus()
			m.inputs[i].PromptStyle = focusedStyle
			m.inputs[i].TextStyle = focusedStyle
		} else {
			m.inputs[i].Blur()
			m.inputs[i].PromptStyle = noStyle
			m.inputs[i].TextStyle = noStyle
		}
	}
}

func (m *SensitiveDataModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m SensitiveDataModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Authentication Required"))
	b.WriteString("\n\n")

	for _, input := range m.inputs {
		b.WriteString(input.View())
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if m.errorMessage != "" {
		b.WriteString(errorStyle.Render("ERROR: " + m.errorMessage))
		b.WriteString("\n\n")
	}

	// Show appropriate help text based on number of fields
	if len(m.inputs) > 1 {
		b.WriteString(helpStyle.Render("Tab/Enter: Next • Shift+Tab: Previous • Esc: Cancel"))
	} else {
		b.WriteString(helpStyle.Render("Enter: Submit • Esc: Cancel"))
	}

	return b.String()
}

func (m SensitiveDataModel) IsFinished() bool {
	return m.finished
}

func (m SensitiveDataModel) IsCancelled() bool {
	return m.cancelled
}

func (m SensitiveDataModel) GetAdminKey() string {
	return m.adminKey
}

func (m SensitiveDataModel) GetKeystorePassphrase() string {
	return m.keystorePass
}

// RunSensitiveDataPrompt runs the sensitive data prompt UI
func RunSensitiveDataPrompt(isAdmin, useE2E bool) (adminKey, keystorePass string, err error) {
	model := NewSensitiveDataPrompt(isAdmin, useE2E)

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return "", "", fmt.Errorf("failed to run sensitive data prompt: %w", err)
	}

	dataModel := finalModel.(SensitiveDataModel)

	if dataModel.IsCancelled() {
		return "", "", fmt.Errorf("authentication cancelled by user")
	}

	if !dataModel.IsFinished() {
		return "", "", fmt.Errorf("authentication not completed")
	}

	return dataModel.GetAdminKey(), dataModel.GetKeystorePassphrase(), nil
}

// RunInteractiveConfig runs the interactive configuration UI
func RunInteractiveConfig() (*Config, string, error) {
	model := NewConfigUI()

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return nil, "", fmt.Errorf("failed to run configuration UI: %w", err)
	}

	configModel := finalModel.(ConfigUIModel)

	if configModel.IsCancelled() {
		return nil, "", fmt.Errorf("configuration cancelled by user")
	}

	if !configModel.IsFinished() {
		return nil, "", fmt.Errorf("configuration not completed")
	}

	return configModel.GetConfig(), configModel.GetKeystorePassphrase(), nil
}
