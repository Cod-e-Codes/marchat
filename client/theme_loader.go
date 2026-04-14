package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	// Optional full-width banner strip (above transcript). Empty uses defaults; info strip falls back to footer colors.
	BannerErrorBg string `json:"banner_error_bg,omitempty"`
	BannerErrorFg string `json:"banner_error_fg,omitempty"`
	BannerWarnBg  string `json:"banner_warn_bg,omitempty"`
	BannerWarnFg  string `json:"banner_warn_fg,omitempty"`
	BannerInfoBg  string `json:"banner_info_bg,omitempty"`
	BannerInfoFg  string `json:"banner_info_fg,omitempty"`
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

// GetCustomThemeNames returns a list of all custom theme names (sorted by key).
// Map iteration order is undefined in Go, so we sort for stable Ctrl+T / :themes order.
func GetCustomThemeNames() []string {
	names := make([]string, 0, len(customThemes))
	for name := range customThemes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ApplyCustomTheme applies a custom theme definition to create theme styles
func ApplyCustomTheme(def ThemeDefinition) themeStyles {
	timeStyle := lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color(def.Colors.Time))
	s := themeStyles{
		User:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(def.Colors.User)),
		Time:       timeStyle,
		Info:       timeStyle,
		Timestamp:  timeStyle,
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
	be, bw, bi := customThemeBannerStrips(def.Colors)
	s.BannerError, s.BannerWarn, s.BannerInfo = be, bw, bi
	s.SystemMsg = lipgloss.NewStyle().Foreground(lipgloss.Color(def.Colors.Message))
	s.SystemMsgError = lipgloss.NewStyle().Foreground(lipgloss.Color(def.Colors.Banner)).Bold(true)
	s.SystemMsgWarn = lipgloss.NewStyle().Foreground(lipgloss.Color(def.Colors.Mention)).Bold(true)
	return s
}

// customThemeBannerStrips builds error, warn, and info banner strip styles from ThemeColors.
func customThemeBannerStrips(c ThemeColors) (lipgloss.Style, lipgloss.Style, lipgloss.Style) {
	errBg, errFg := strings.TrimSpace(c.BannerErrorBg), strings.TrimSpace(c.BannerErrorFg)
	if errBg == "" {
		errBg = "#C42B2B"
	}
	if errFg == "" {
		errFg = "#FFFFFF"
	}
	warnBg, warnFg := strings.TrimSpace(c.BannerWarnBg), strings.TrimSpace(c.BannerWarnFg)
	if warnBg == "" {
		warnBg = "#B8860B"
	}
	if warnFg == "" {
		warnFg = "#000000"
	}
	infoBg, infoFg := strings.TrimSpace(c.BannerInfoBg), strings.TrimSpace(c.BannerInfoFg)
	if infoBg == "" {
		infoBg = strings.TrimSpace(c.FooterBg)
		if infoBg == "" {
			infoBg = "#2A2A2A"
		}
	}
	if infoFg == "" {
		infoFg = strings.TrimSpace(c.FooterFg)
		if infoFg == "" {
			infoFg = "#CCCCCC"
		}
	}
	errStrip := lipgloss.NewStyle().Background(lipgloss.Color(errBg)).Foreground(lipgloss.Color(errFg)).Bold(true)
	warnStrip := lipgloss.NewStyle().Background(lipgloss.Color(warnBg)).Foreground(lipgloss.Color(warnFg)).Bold(true)
	infoStrip := lipgloss.NewStyle().Background(lipgloss.Color(infoBg)).Foreground(lipgloss.Color(infoFg)).Bold(true)
	return errStrip, warnStrip, infoStrip
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
