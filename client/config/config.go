// Enhanced config.go for the client
package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Config struct {
	// Connection settings
	Username  string `json:"username"`
	ServerURL string `json:"server_url"`

	// Admin settings (optional)
	IsAdmin  bool   `json:"is_admin,omitempty"`
	AdminKey string `json:"admin_key,omitempty"` // Note: Consider security implications

	// E2E Encryption settings (optional)
	UseE2E bool `json:"use_e2e,omitempty"`

	// UI settings
	Theme          string `json:"theme"`
	TwentyFourHour bool   `json:"twenty_four_hour"`
	SkipTLSVerify  bool   `json:"skip_tls_verify,omitempty"`

	// Bell notification settings (legacy - kept for backward compatibility)
	EnableBell    bool `json:"enable_bell,omitempty"`     // Enable/disable bell
	BellOnMention bool `json:"bell_on_mention,omitempty"` // Only bell on mentions

	// Enhanced notification settings
	NotificationMode     string `json:"notification_mode,omitempty"`     // "none", "bell", "desktop", "both"
	DesktopNotifications bool   `json:"desktop_notifications,omitempty"` // Enable desktop notifications
	DesktopOnMention     bool   `json:"desktop_on_mention,omitempty"`    // Desktop notify on mentions
	DesktopOnDM          bool   `json:"desktop_on_dm,omitempty"`         // Desktop notify on DMs
	DesktopOnAll         bool   `json:"desktop_on_all,omitempty"`        // Desktop notify on all messages
	QuietHoursEnabled    bool   `json:"quiet_hours_enabled,omitempty"`   // Enable quiet hours
	QuietHoursStart      int    `json:"quiet_hours_start,omitempty"`     // Quiet hours start (hour 0-23)
	QuietHoursEnd        int    `json:"quiet_hours_end,omitempty"`       // Quiet hours end (hour 0-23)

	// Quick start settings
	SaveCredentials bool  `json:"save_credentials"`
	LastUsed        int64 `json:"last_used,omitempty"`
}

// ConnectionProfile represents a saved connection profile
type ConnectionProfile struct {
	Name      string `json:"name"`
	ServerURL string `json:"server_url"`
	Username  string `json:"username"`
	IsAdmin   bool   `json:"is_admin"`
	UseE2E    bool   `json:"use_e2e"`
	Theme     string `json:"theme,omitempty"`
	LastUsed  int64  `json:"last_used,omitempty"` // Unix timestamp
}

type Profiles struct {
	Default  string              `json:"default,omitempty"`
	Profiles []ConnectionProfile `json:"profiles"`
}

// InteractiveConfigLoader handles interactive configuration
type InteractiveConfigLoader struct {
	ConfigPath   string
	ProfilesPath string
	reader       *bufio.Reader
}

func NewInteractiveConfigLoader() (*InteractiveConfigLoader, error) {
	if err := EnsureClientConfigDir(); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}
	configDir := ResolveClientConfigDir()

	return &InteractiveConfigLoader{
		ConfigPath:   filepath.Join(configDir, "config.json"),
		ProfilesPath: filepath.Join(configDir, "profiles.json"),
		reader:       bufio.NewReader(os.Stdin),
	}, nil
}

// LoadOrPromptConfig loads existing config or prompts for new configuration
func (icl *InteractiveConfigLoader) LoadOrPromptConfig(overrides map[string]interface{}) (*Config, string, string, string, error) {
	// Try to load existing config first
	cfg, err := LoadConfig(icl.ConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, "", "", "", fmt.Errorf("error loading config: %w", err)
	}

	// Load profiles
	profiles, err := icl.LoadProfiles()
	if err != nil && !os.IsNotExist(err) {
		return nil, "", "", "", fmt.Errorf("error loading profiles: %w", err)
	}

	// Show welcome message
	fmt.Println("Welcome to marchat!")
	fmt.Println()

	// Check if user wants to use a profile or create new connection
	if len(profiles.Profiles) > 0 {
		// Use enhanced profile selection with management features
		selectedProfile, isCreateNew, err := RunEnhancedProfileSelectionWithNew(profiles.Profiles, icl)
		if err != nil {
			return nil, "", "", "", err
		}

		if !isCreateNew {
			// We now have the actual profile object, not just an index!
			// Reload profiles in case they were modified during selection
			profiles, err = icl.LoadProfiles()
			if err != nil {
				return nil, "", "", "", fmt.Errorf("error reloading profiles: %w", err)
			}

			// Find the selected profile in the reloaded list
			var profileIndex = -1
			for i, p := range profiles.Profiles {
				if p.Name == selectedProfile.Name &&
					p.ServerURL == selectedProfile.ServerURL &&
					p.Username == selectedProfile.Username {
					profileIndex = i
					break
				}
			}

			if profileIndex == -1 {
				return nil, "", "", "", fmt.Errorf("selected profile no longer exists")
			}

			// Now use the correct profile
			profile := profiles.Profiles[profileIndex]
			cfg = icl.profileToConfig(profile)

			// Update last used timestamp
			profile.LastUsed = time.Now().Unix()
			profiles.Profiles[profileIndex] = profile
			if err := icl.SaveProfiles(profiles); err != nil {
				fmt.Printf("Warning: Could not update profile usage: %v\n", err)
			}

			// Still need to prompt for sensitive data
			adminKey, keystorePass, err := icl.promptSensitiveData(cfg.IsAdmin, cfg.UseE2E)
			if err != nil {
				return nil, "", "", "", err
			}

			return &cfg, icl.formatLaunchCommand(&cfg, adminKey, keystorePass), adminKey, keystorePass, nil
		}
		// If isCreateNew is true, continue to create new configuration below
	}

	// Create new configuration interactively
	newCfg, err := icl.promptNewConfig()
	if err != nil {
		return nil, "", "", "", err
	}

	// Apply command line overrides
	icl.applyOverrides(newCfg, overrides)

	// Ask if user wants to save this as a profile
	if icl.promptYesNo("Save this connection as a profile for future use?", true) {
		profileName, err := icl.promptString("Profile name", fmt.Sprintf("%s@%s", newCfg.Username, newCfg.ServerURL))
		if err != nil {
			return nil, "", "", "", err
		}

		profile := ConnectionProfile{
			Name:      profileName,
			ServerURL: newCfg.ServerURL,
			Username:  newCfg.Username,
			IsAdmin:   newCfg.IsAdmin,
			UseE2E:    newCfg.UseE2E,
			Theme:     newCfg.Theme,
		}

		if err := icl.saveProfile(profile); err != nil {
			fmt.Printf("Could not save profile: %v\n", err)
		} else {
			fmt.Printf("Profile '%s' saved!\n", profileName)
		}
	}

	// Save basic config (without sensitive data)
	if err := SaveConfig(icl.ConfigPath, *newCfg); err != nil {
		fmt.Printf("Could not save config: %v\n", err)
	}

	// Prompt for sensitive data
	adminKey, keystorePass, err := icl.promptSensitiveData(newCfg.IsAdmin, newCfg.UseE2E)
	if err != nil {
		return nil, "", "", "", err
	}

	return newCfg, icl.formatLaunchCommand(newCfg, adminKey, keystorePass), adminKey, keystorePass, nil
}

func (icl *InteractiveConfigLoader) promptNewConfig() (*Config, error) {
	cfg := &Config{
		Theme:          "system",
		TwentyFourHour: true,
	}

	// Server URL
	serverURL, err := icl.promptString("Server URL", "")
	if err != nil {
		return nil, err
	}
	cfg.ServerURL = serverURL

	// Username
	username, err := icl.promptString("Username", "")
	if err != nil {
		return nil, err
	}
	cfg.Username = username

	// Admin privileges
	cfg.IsAdmin = icl.promptYesNo("Connect as admin?", false)

	// E2E encryption
	cfg.UseE2E = icl.promptYesNo("Enable end-to-end encryption?", false)

	// TLS verification
	if strings.HasPrefix(cfg.ServerURL, "wss://") {
		cfg.SkipTLSVerify = icl.promptYesNo("Skip TLS certificate verification? (only for development)", false)
	}

	// Theme
	fmt.Println("\nAvailable themes: system, patriot, retro, modern")
	fmt.Println("(Custom themes can be defined in themes.json)")
	theme, err := icl.promptString("Theme", "system")
	if err != nil {
		return nil, err
	}
	cfg.Theme = theme

	return cfg, nil
}

func (icl *InteractiveConfigLoader) promptSensitiveData(isAdmin, useE2E bool) (adminKey, keystorePass string, err error) {
	// Use Bubble Tea UI for consistent user experience
	return RunSensitiveDataPrompt(isAdmin, useE2E)
}

func (icl *InteractiveConfigLoader) promptString(prompt, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	// Use a more reliable method for reading input on Windows
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		// If Scanln fails, try the original method as fallback
		response, err = icl.reader.ReadString('\n')
		if err != nil {
			return "", err
		}
	}

	response = strings.TrimSpace(response)
	if response == "" && defaultValue != "" {
		return defaultValue, nil
	}

	return response, nil
}

func (icl *InteractiveConfigLoader) promptYesNo(prompt string, defaultValue bool) bool {
	defaultStr := "y/N"
	if defaultValue {
		defaultStr = "Y/n"
	}

	fmt.Printf("%s [%s]: ", prompt, defaultStr)

	// Use a more reliable method for reading input on Windows
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		// If Scanln fails, try the original method as fallback
		response, err = icl.reader.ReadString('\n')
		if err != nil {
			return defaultValue
		}
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "" {
		return defaultValue
	}

	return response == "y" || response == "yes"
}

func (icl *InteractiveConfigLoader) LoadProfiles() (*Profiles, error) {
	var profiles Profiles

	if _, err := os.Stat(icl.ProfilesPath); os.IsNotExist(err) {
		return &profiles, nil
	}

	data, err := os.ReadFile(icl.ProfilesPath)
	if err != nil {
		return nil, err
	}

	// Handle empty file
	if len(data) == 0 {
		return &profiles, nil
	}

	if err := json.Unmarshal(data, &profiles); err != nil {
		return nil, err
	}

	return &profiles, nil
}

func (icl *InteractiveConfigLoader) saveProfile(profile ConnectionProfile) error {
	profiles, err := icl.LoadProfiles()
	if err != nil {
		profiles = &Profiles{}
	}

	// Check if profile with same name exists
	for i, existing := range profiles.Profiles {
		if existing.Name == profile.Name {
			// Update existing profile
			profiles.Profiles[i] = profile
			return icl.SaveProfiles(profiles)
		}
	}

	// Add new profile
	profiles.Profiles = append(profiles.Profiles, profile)
	return icl.SaveProfiles(profiles)
}

func (icl *InteractiveConfigLoader) SaveProfiles(profiles *Profiles) error {
	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(icl.ProfilesPath, data, 0600)
}

func (icl *InteractiveConfigLoader) profileToConfig(profile ConnectionProfile) Config {
	return Config{
		Username:       profile.Username,
		ServerURL:      profile.ServerURL,
		IsAdmin:        profile.IsAdmin,
		UseE2E:         profile.UseE2E,
		Theme:          profile.Theme,
		TwentyFourHour: true, // Default
	}
}

func (icl *InteractiveConfigLoader) applyOverrides(cfg *Config, overrides map[string]interface{}) {
	if val, ok := overrides["server"]; ok {
		if str, ok := val.(string); ok && str != "" {
			cfg.ServerURL = str
		}
	}
	if val, ok := overrides["username"]; ok {
		if str, ok := val.(string); ok && str != "" {
			cfg.Username = str
		}
	}
	if val, ok := overrides["admin"]; ok {
		if b, ok := val.(bool); ok {
			cfg.IsAdmin = b
		}
	}
	if val, ok := overrides["e2e"]; ok {
		if b, ok := val.(bool); ok {
			cfg.UseE2E = b
		}
	}
	if val, ok := overrides["theme"]; ok {
		if str, ok := val.(string); ok && str != "" {
			cfg.Theme = str
		}
	}
	if val, ok := overrides["skip-tls-verify"]; ok {
		if b, ok := val.(bool); ok {
			cfg.SkipTLSVerify = b
		}
	}
}

func (icl *InteractiveConfigLoader) formatLaunchCommand(cfg *Config, adminKey, keystorePass string) string {
	var parts []string

	parts = append(parts, "./marchat-client")
	parts = append(parts, fmt.Sprintf("--server %s", cfg.ServerURL))
	parts = append(parts, fmt.Sprintf("--username %s", cfg.Username))

	if cfg.IsAdmin {
		parts = append(parts, "--admin")
		parts = append(parts, fmt.Sprintf("--admin-key %s", adminKey))
	}

	if cfg.UseE2E {
		parts = append(parts, "--e2e")
		parts = append(parts, fmt.Sprintf("--keystore-passphrase %s", keystorePass))
	}

	if cfg.SkipTLSVerify {
		parts = append(parts, "--skip-tls-verify")
	}

	if cfg.Theme != "system" {
		parts = append(parts, fmt.Sprintf("--theme %s", cfg.Theme))
	}

	return strings.Join(parts, " ")
}

// FormatSanitizedLaunchCommand creates a version safe for logging (without sensitive data)
func (icl *InteractiveConfigLoader) FormatSanitizedLaunchCommand(cfg *Config) string {
	var parts []string

	parts = append(parts, "./marchat-client")
	parts = append(parts, fmt.Sprintf("--server %s", cfg.ServerURL))
	parts = append(parts, fmt.Sprintf("--username %s", cfg.Username))

	if cfg.IsAdmin {
		parts = append(parts, "--admin")
		parts = append(parts, "--admin-key <your-admin-key>")
	}

	if cfg.UseE2E {
		parts = append(parts, "--e2e")
		parts = append(parts, "--keystore-passphrase <your-passphrase>")
	}

	if cfg.SkipTLSVerify {
		parts = append(parts, "--skip-tls-verify")
	}

	if cfg.Theme != "system" {
		parts = append(parts, fmt.Sprintf("--theme %s", cfg.Theme))
	}

	return strings.Join(parts, " ")
}

// AutoConnect automatically connects to the most recently used profile
func (icl *InteractiveConfigLoader) AutoConnect() (*Config, error) {
	profiles, err := icl.LoadProfiles()
	if err != nil {
		return nil, err
	}

	if len(profiles.Profiles) == 0 {
		return nil, fmt.Errorf("no saved profiles found - run without --auto to create one")
	}

	// Find most recently used profile
	var mostRecent *ConnectionProfile
	var mostRecentTime int64

	for i, profile := range profiles.Profiles {
		if profile.LastUsed > mostRecentTime {
			mostRecentTime = profile.LastUsed
			mostRecent = &profiles.Profiles[i]
		}
	}

	if mostRecent == nil {
		// No usage timestamps, use first profile
		mostRecent = &profiles.Profiles[0]
	}

	fmt.Printf("Auto-connecting to: %s (%s@%s)\n", mostRecent.Name, mostRecent.Username, mostRecent.ServerURL)

	// Update last used timestamp
	mostRecent.LastUsed = time.Now().Unix()
	if err := icl.SaveProfiles(profiles); err != nil {
		// Log error but don't fail the connection
		fmt.Printf("Warning: Could not update profile usage timestamp: %v\n", err)
	}

	cfg := icl.profileToConfig(*mostRecent)
	return &cfg, nil
}

// QuickStartConnect shows profiles with management features and connects to selected one
func (icl *InteractiveConfigLoader) QuickStartConnect() (*Config, error) {
	profiles, err := icl.LoadProfiles()
	if err != nil {
		return nil, err
	}

	if len(profiles.Profiles) == 0 {
		fmt.Println("No saved connection profiles found.")
		fmt.Println("Run without --quick-start to create your first connection profile.")
		return nil, fmt.Errorf("no saved profiles")
	}

	// Sort profiles by last used (most recent first)
	sort.Slice(profiles.Profiles, func(i, j int) bool {
		return profiles.Profiles[i].LastUsed > profiles.Profiles[j].LastUsed
	})

	// Use enhanced Bubble Tea UI for profile selection with management features
	selectedProfile, err := RunEnhancedProfileSelection(profiles.Profiles, icl)
	if err != nil {
		return nil, err
	}

	// We now have the actual profile object, not just an index!
	// Reload profiles in case they were modified
	profiles, err = icl.LoadProfiles()
	if err != nil {
		return nil, err
	}

	// Find the selected profile in the reloaded list
	var profileIndex = -1
	for i, p := range profiles.Profiles {
		if p.Name == selectedProfile.Name &&
			p.ServerURL == selectedProfile.ServerURL &&
			p.Username == selectedProfile.Username {
			profileIndex = i
			break
		}
	}

	if profileIndex == -1 {
		return nil, fmt.Errorf("selected profile no longer exists")
	}

	profile := &profiles.Profiles[profileIndex]
	fmt.Printf("Selected: %s\n", profile.Name)

	// Update last used timestamp
	profile.LastUsed = time.Now().Unix()
	if err := icl.SaveProfiles(profiles); err != nil {
		// Log error but don't fail the connection
		fmt.Printf("Warning: Could not update profile usage timestamp: %v\n", err)
	}

	cfg := icl.profileToConfig(*profile)
	return &cfg, nil
}

// PromptSensitiveData prompts for admin key and/or keystore passphrase
func (icl *InteractiveConfigLoader) PromptSensitiveData(isAdmin, useE2E bool) (adminKey, keystorePass string, err error) {
	return icl.promptSensitiveData(isAdmin, useE2E)
}

// ResolveClientConfigDir returns the client configuration directory using the same
// rules as the marchat client binary: MARCHAT_CONFIG_DIR if set, otherwise the
// per-user application data directory from GetConfigDir (same when developing from
// a git clone or using a release binary). The server uses ./config in-repo for .env
// and the database; the client does not use that path unless you set MARCHAT_CONFIG_DIR.
func ResolveClientConfigDir() string {
	if envConfigDir := os.Getenv("MARCHAT_CONFIG_DIR"); envConfigDir != "" {
		return envConfigDir
	}
	dir, err := GetConfigDir()
	if err != nil {
		return "./config"
	}
	return dir
}

// EnsureClientConfigDir creates the resolved client config directory if needed.
func EnsureClientConfigDir() error {
	return os.MkdirAll(ResolveClientConfigDir(), 0755)
}

// GetConfigDir returns the platform-appropriate configuration directory
func GetConfigDir() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		// Windows: %APPDATA%\marchat
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", os.ErrNotExist
		}
		configDir = filepath.Join(appData, "marchat")
	case "darwin":
		// macOS: ~/Library/Application Support/marchat
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, "Library", "Application Support", "marchat")
	case "linux", "android":
		// Linux and Android/Termux: ~/.config/marchat
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".config", "marchat")
	default:
		// Fallback for other platforms
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".config", "marchat")
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return configDir, nil
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

// GetKeystorePath returns the full path to the keystore file
// Checks old location (current directory) first for backward compatibility
func GetKeystorePath() (string, error) {
	// Check if keystore exists in current directory (legacy location)
	legacyPath := "keystore.dat"
	if _, err := os.Stat(legacyPath); err == nil {
		// Legacy keystore exists, use it
		abs, err := filepath.Abs(legacyPath)
		if err != nil {
			return legacyPath, nil // fallback to relative path
		}
		return abs, nil
	}

	// Prefer keystore next to config.json (same rules as ResolveClientConfigDir).
	primary := filepath.Join(ResolveClientConfigDir(), "keystore.dat")
	if _, err := os.Stat(primary); err == nil {
		return filepath.Abs(primary)
	}

	// Older builds stored the keystore under GetConfigDir() even when running from a repo.
	userDir, err := GetConfigDir()
	if err != nil {
		return filepath.Abs(primary)
	}
	fallback := filepath.Join(userDir, "keystore.dat")
	if _, err := os.Stat(fallback); err == nil {
		return filepath.Abs(fallback)
	}

	return filepath.Abs(primary)
}

// MigrateKeystoreToNewLocation migrates keystore from legacy location to new platform-appropriate location
func MigrateKeystoreToNewLocation() error {
	legacyPath := "keystore.dat"

	// Check if legacy keystore exists
	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		return nil // No legacy keystore to migrate
	}

	newPath := filepath.Join(ResolveClientConfigDir(), "keystore.dat")
	if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
		return err
	}

	// Check if new keystore already exists
	if _, err := os.Stat(newPath); err == nil {
		return nil // New keystore already exists, don't overwrite
	}

	// Copy legacy keystore to new location
	input, err := os.ReadFile(legacyPath)
	if err != nil {
		return err
	}

	if err := os.WriteFile(newPath, input, 0600); err != nil {
		return err
	}

	return nil
}

func LoadConfig(path string) (Config, error) {
	var cfg Config
	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func SaveConfig(path string, cfg Config) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(cfg)
}
