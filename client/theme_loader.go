package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ThemeColors defines all customizable colors for a theme
type ThemeColors struct {
	User              string `json:"user"`
	Time              string `json:"time"`
	Message           string `json:"message"`
	Banner            string `json:"banner"`
	BoxBorder         string `json:"box_border"`
	Mention           string `json:"mention"`
	Hyperlink         string `json:"hyperlink"`
	UserListBorder    string `json:"user_list_border"`
	Me                string `json:"me"`
	Other             string `json:"other"`
	Background        string `json:"background"`
	HeaderBg          string `json:"header_bg"`
	HeaderFg          string `json:"header_fg"`
	FooterBg          string `json:"footer_bg"`
	FooterFg          string `json:"footer_fg"`
	InputBg           string `json:"input_bg"`
	InputFg           string `json:"input_fg"`
	HelpOverlayBg     string `json:"help_overlay_bg"`
	HelpOverlayFg     string `json:"help_overlay_fg"`
	HelpOverlayBorder string `json:"help_overlay_border"`
	HelpTitle         string `json:"help_title"`
}

// ThemeDefinition represents a complete theme with metadata
type ThemeDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Colors      ThemeColors `json:"colors"`
}

// ThemeFile represents the structure of themes.json
type ThemeFile map[string]ThemeDefinition

var customThemes ThemeFile

// LoadCustomThemes loads custom themes from the themes.json file
func LoadCustomThemes() error {
	// Try multiple locations in order of preference
	locations := []string{
		"themes.json", // Current directory
		filepath.Join(getClientConfigDir(), "themes.json"), // Config directory
	}

	var data []byte
	var err error
	var foundPath string

	for _, path := range locations {
		data, err = os.ReadFile(path)
		if err == nil {
			foundPath = path
			break
		}
	}

	if err != nil {
		// No custom themes file found - this is OK, we'll use built-in themes
		customThemes = make(ThemeFile)
		return nil
	}

	if err := json.Unmarshal(data, &customThemes); err != nil {
		return fmt.Errorf("failed to parse %s: %w", foundPath, err)
	}

	return nil
}

// GetCustomThemeNames returns a list of all custom theme names
func GetCustomThemeNames() []string {
	names := make([]string, 0, len(customThemes))
	for name := range customThemes {
		names = append(names, name)
	}
	return names
}

// ApplyCustomTheme applies a custom theme definition to create theme styles
func ApplyCustomTheme(def ThemeDefinition) themeStyles {
	s := themeStyles{
		User:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(def.Colors.User)),
		Time:       lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color(def.Colors.Time)),
		Msg:        lipgloss.NewStyle().Foreground(lipgloss.Color(def.Colors.Message)),
		Banner:     lipgloss.NewStyle().Foreground(lipgloss.Color(def.Colors.Banner)).Bold(true),
		Box:        lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color(def.Colors.BoxBorder)),
		Mention:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(def.Colors.Mention)),
		Hyperlink:  lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color(def.Colors.Hyperlink)),
		UserList:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(def.Colors.UserListBorder)).Padding(0, 1),
		Me:         lipgloss.NewStyle().Foreground(lipgloss.Color(def.Colors.Me)).Bold(true),
		Other:      lipgloss.NewStyle().Foreground(lipgloss.Color(def.Colors.Other)),
		Background: lipgloss.NewStyle().Background(lipgloss.Color(def.Colors.Background)),
		Header:     lipgloss.NewStyle().Background(lipgloss.Color(def.Colors.HeaderBg)).Foreground(lipgloss.Color(def.Colors.HeaderFg)).Bold(true),
		Footer:     lipgloss.NewStyle().Background(lipgloss.Color(def.Colors.FooterBg)).Foreground(lipgloss.Color(def.Colors.FooterFg)),
		Input:      lipgloss.NewStyle().Background(lipgloss.Color(def.Colors.InputBg)).Foreground(lipgloss.Color(def.Colors.InputFg)),
		HelpOverlay: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(def.Colors.HelpOverlayBorder)).
			Background(lipgloss.Color(def.Colors.HelpOverlayBg)).
			Foreground(lipgloss.Color(def.Colors.HelpOverlayFg)).
			Padding(1, 2),
		HelpTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(def.Colors.HelpTitle)).
			Bold(true).
			MarginBottom(1),
	}
	return s
}

// IsCustomTheme checks if a theme name refers to a custom theme
func IsCustomTheme(themeName string) bool {
	_, exists := customThemes[themeName]
	return exists
}

// GetCustomTheme retrieves a custom theme by name
func GetCustomTheme(themeName string) (ThemeDefinition, bool) {
	theme, exists := customThemes[themeName]
	return theme, exists
}

// GetCustomThemeByName performs case-insensitive lookup of custom themes by display name
func GetCustomThemeByName(displayName string) (string, ThemeDefinition, bool) {
	displayNameLower := strings.ToLower(displayName)

	// First try exact key match
	if theme, exists := customThemes[displayNameLower]; exists {
		return displayNameLower, theme, true
	}

	// Then try case-insensitive key match
	for key, theme := range customThemes {
		if strings.ToLower(key) == displayNameLower {
			return key, theme, true
		}
	}

	// Finally try case-insensitive display name match
	for key, theme := range customThemes {
		if strings.ToLower(theme.Name) == displayNameLower {
			return key, theme, true
		}
	}

	return "", ThemeDefinition{}, false
}

// ListAllThemes returns all available themes (built-in + custom)
func ListAllThemes() []string {
	builtIn := []string{"system", "patriot", "retro", "modern"}
	custom := GetCustomThemeNames()
	return append(builtIn, custom...)
}

// GetThemeInfo returns human-readable information about a theme
func GetThemeInfo(themeName string) string {
	// Check for built-in themes
	builtInDescriptions := map[string]string{
		"system":  "Uses terminal's default colors",
		"patriot": "American patriotic theme (red, white, blue)",
		"retro":   "Retro terminal theme (orange, green)",
		"modern":  "Modern dark blue-gray theme",
	}

	if desc, ok := builtInDescriptions[themeName]; ok {
		return fmt.Sprintf("%s (built-in): %s", themeName, desc)
	}

	// Check for custom themes
	if theme, ok := customThemes[themeName]; ok {
		if theme.Description != "" {
			return fmt.Sprintf("%s: %s", theme.Name, theme.Description)
		}
		return fmt.Sprintf("%s (custom theme)", theme.Name)
	}

	return fmt.Sprintf("%s (unknown theme)", themeName)
}
