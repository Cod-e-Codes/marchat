package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write test config
	configData := `{
		"username": "testuser",
		"server_url": "ws://localhost:8080",
		"is_admin": true,
		"theme": "modern",
		"twenty_four_hour": true,
		"enable_bell": false
	}`

	err := os.WriteFile(configPath, []byte(configData), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values
	if cfg.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", cfg.Username)
	}

	if cfg.ServerURL != "ws://localhost:8080" {
		t.Errorf("Expected server_url 'ws://localhost:8080', got '%s'", cfg.ServerURL)
	}

	if !cfg.IsAdmin {
		t.Error("Expected is_admin to be true")
	}

	if cfg.Theme != "modern" {
		t.Errorf("Expected theme 'modern', got '%s'", cfg.Theme)
	}

	if !cfg.TwentyFourHour {
		t.Error("Expected twenty_four_hour to be true")
	}

	if cfg.EnableBell {
		t.Error("Expected enable_bell to be false")
	}
}

func TestLoadConfigFileNotExist(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.json")
	if err == nil {
		t.Error("Expected error when loading nonexistent config file")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	// Create temporary file with invalid JSON
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	err := os.WriteFile(configPath, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error when loading invalid JSON")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := Config{
		Username:        "testuser",
		ServerURL:       "ws://localhost:8080",
		IsAdmin:         true,
		Theme:           "modern",
		TwentyFourHour:  true,
		EnableBell:      false,
		SaveCredentials: true,
	}

	// Save config
	err := SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Expected config file to be created")
	}

	// Load and verify
	loadedCfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedCfg.Username != cfg.Username {
		t.Errorf("Expected username %s, got %s", cfg.Username, loadedCfg.Username)
	}

	if loadedCfg.ServerURL != cfg.ServerURL {
		t.Errorf("Expected server_url %s, got %s", cfg.ServerURL, loadedCfg.ServerURL)
	}

	if loadedCfg.IsAdmin != cfg.IsAdmin {
		t.Errorf("Expected is_admin %v, got %v", cfg.IsAdmin, loadedCfg.IsAdmin)
	}
}

func TestGetConfigDir(t *testing.T) {
	configDir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("Failed to get config dir: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("Expected config directory to exist")
	}

	// Verify platform-specific path
	var expectedBase string
	switch runtime.GOOS {
	case "windows":
		expectedBase = "marchat"
	case "darwin":
		expectedBase = "marchat"
	case "linux", "android":
		expectedBase = "marchat"
	default:
		expectedBase = "marchat"
	}

	if filepath.Base(configDir) != expectedBase {
		t.Errorf("Expected config dir to end with '%s', got '%s'", expectedBase, filepath.Base(configDir))
	}
}

func TestGetConfigPath(t *testing.T) {
	configPath, err := GetConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	// Verify path ends with config.json
	if filepath.Base(configPath) != "config.json" {
		t.Errorf("Expected config path to end with 'config.json', got '%s'", filepath.Base(configPath))
	}

	// Verify parent directory exists
	parentDir := filepath.Dir(configPath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		t.Error("Expected config directory to exist")
	}
}

func TestGetKeystorePath(t *testing.T) {
	// Test with no legacy keystore
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore original directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	keystorePath, err := GetKeystorePath()
	if err != nil {
		t.Fatalf("Failed to get keystore path: %v", err)
	}

	// Should return platform-appropriate path
	if filepath.Base(keystorePath) != "keystore.dat" {
		t.Errorf("Expected keystore path to end with 'keystore.dat', got '%s'", filepath.Base(keystorePath))
	}
}

func TestGetKeystorePathWithLegacy(t *testing.T) {
	// Test with legacy keystore in current directory
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore original directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create legacy keystore
	legacyPath := "keystore.dat"
	err := os.WriteFile(legacyPath, []byte("legacy keystore"), 0644)
	if err != nil {
		t.Fatalf("Failed to create legacy keystore: %v", err)
	}

	keystorePath, err := GetKeystorePath()
	if err != nil {
		t.Fatalf("Failed to get keystore path: %v", err)
	}

	// Should return legacy path
	absLegacyPath, _ := filepath.Abs(legacyPath)
	if keystorePath != absLegacyPath {
		t.Errorf("Expected legacy keystore path %s, got %s", absLegacyPath, keystorePath)
	}
}

func TestMigrateKeystoreToNewLocation(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore original directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Clean up any existing keystore in the resolved config directory
	newPath := filepath.Join(ResolveClientConfigDir(), "keystore.dat")
	os.Remove(newPath) // Ignore error if file doesn't exist

	// Test with no legacy keystore
	err := MigrateKeystoreToNewLocation()
	if err != nil {
		t.Fatalf("Expected no error when no legacy keystore exists: %v", err)
	}

	// Create legacy keystore
	legacyData := []byte("legacy keystore data")
	err = os.WriteFile("keystore.dat", legacyData, 0644)
	if err != nil {
		t.Fatalf("Failed to create legacy keystore: %v", err)
	}

	// Migrate
	err = MigrateKeystoreToNewLocation()
	if err != nil {
		t.Fatalf("Failed to migrate keystore: %v", err)
	}

	// Verify new keystore exists
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Error("Expected new keystore to be created")
	}

	// Verify content matches
	newData, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("Failed to read new keystore: %v", err)
	}

	if string(newData) != string(legacyData) {
		t.Errorf("New keystore content does not match legacy keystore. Expected: %s, Got: %s", string(legacyData), string(newData))
	}
}

func TestMigrateKeystoreAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore original directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create legacy keystore
	legacyData := []byte("legacy keystore data")
	err := os.WriteFile("keystore.dat", legacyData, 0644)
	if err != nil {
		t.Fatalf("Failed to create legacy keystore: %v", err)
	}

	// Create new keystore first (same target as ResolveClientConfigDir in this cwd)
	newPath := filepath.Join(ResolveClientConfigDir(), "keystore.dat")
	err = os.WriteFile(newPath, []byte("existing new keystore"), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing new keystore: %v", err)
	}

	// Try to migrate
	err = MigrateKeystoreToNewLocation()
	if err != nil {
		t.Fatalf("Expected no error when new keystore already exists: %v", err)
	}

	// Verify existing new keystore was not overwritten
	newData, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("Failed to read new keystore: %v", err)
	}

	if string(newData) != "existing new keystore" {
		t.Error("Existing new keystore should not be overwritten")
	}
}

func TestInteractiveConfigLoaderProfileToConfig(t *testing.T) {
	icl := &InteractiveConfigLoader{}

	profile := ConnectionProfile{
		Name:      "test-profile",
		ServerURL: "ws://test:8080",
		Username:  "testuser",
		IsAdmin:   true,
		UseE2E:    true,
		Theme:     "modern",
		LastUsed:  time.Now().Unix(),
	}

	cfg := icl.profileToConfig(profile)

	if cfg.Username != profile.Username {
		t.Errorf("Expected username %s, got %s", profile.Username, cfg.Username)
	}

	if cfg.ServerURL != profile.ServerURL {
		t.Errorf("Expected server_url %s, got %s", profile.ServerURL, cfg.ServerURL)
	}

	if cfg.IsAdmin != profile.IsAdmin {
		t.Errorf("Expected is_admin %v, got %v", profile.IsAdmin, cfg.IsAdmin)
	}

	if cfg.UseE2E != profile.UseE2E {
		t.Errorf("Expected use_e2e %v, got %v", profile.UseE2E, cfg.UseE2E)
	}

	if cfg.Theme != profile.Theme {
		t.Errorf("Expected theme %s, got %s", profile.Theme, cfg.Theme)
	}

	if !cfg.TwentyFourHour {
		t.Error("Expected twenty_four_hour to be true by default")
	}
}

func TestInteractiveConfigLoaderApplyOverrides(t *testing.T) {
	cfg := &Config{
		Username:      "original",
		ServerURL:     "ws://original:8080",
		IsAdmin:       false,
		UseE2E:        false,
		Theme:         "system",
		SkipTLSVerify: false,
	}

	icl := &InteractiveConfigLoader{}

	overrides := map[string]interface{}{
		"server":          "ws://override:8080",
		"username":        "override",
		"admin":           true,
		"e2e":             true,
		"theme":           "modern",
		"skip-tls-verify": true,
	}

	icl.applyOverrides(cfg, overrides)

	if cfg.ServerURL != "ws://override:8080" {
		t.Errorf("Expected server_url to be overridden, got %s", cfg.ServerURL)
	}

	if cfg.Username != "override" {
		t.Errorf("Expected username to be overridden, got %s", cfg.Username)
	}

	if !cfg.IsAdmin {
		t.Error("Expected is_admin to be overridden to true")
	}

	if !cfg.UseE2E {
		t.Error("Expected use_e2e to be overridden to true")
	}

	if cfg.Theme != "modern" {
		t.Errorf("Expected theme to be overridden, got %s", cfg.Theme)
	}

	if !cfg.SkipTLSVerify {
		t.Error("Expected skip_tls_verify to be overridden to true")
	}
}

func TestInteractiveConfigLoaderFormatLaunchCommand(t *testing.T) {
	tmpDir := t.TempDir()
	icl := &InteractiveConfigLoader{
		ConfigPath:   filepath.Join(tmpDir, "config.json"),
		ProfilesPath: filepath.Join(tmpDir, "profiles.json"),
	}

	cfg := &Config{
		Username:      "testuser",
		ServerURL:     "ws://localhost:8080",
		IsAdmin:       true,
		UseE2E:        true,
		SkipTLSVerify: true,
		Theme:         "modern",
	}

	command := icl.formatLaunchCommand(cfg, "admin-key", "keystore-pass")

	// Verify command contains expected parts
	if !contains(command, "--server ws://localhost:8080") {
		t.Error("Expected command to contain server URL")
	}

	if !contains(command, "--username testuser") {
		t.Error("Expected command to contain username")
	}

	if !contains(command, "--admin") {
		t.Error("Expected command to contain --admin flag")
	}

	if !contains(command, "--admin-key admin-key") {
		t.Error("Expected command to contain admin key")
	}

	if !contains(command, "--e2e") {
		t.Error("Expected command to contain --e2e flag")
	}

	if !contains(command, "--keystore-passphrase keystore-pass") {
		t.Error("Expected command to contain keystore passphrase")
	}

	if !contains(command, "--skip-tls-verify") {
		t.Error("Expected command to contain --skip-tls-verify flag")
	}

	if !contains(command, "--theme modern") {
		t.Error("Expected command to contain theme")
	}
}

func TestInteractiveConfigLoaderFormatSanitizedLaunchCommand(t *testing.T) {
	tmpDir := t.TempDir()
	icl := &InteractiveConfigLoader{
		ConfigPath:   filepath.Join(tmpDir, "config.json"),
		ProfilesPath: filepath.Join(tmpDir, "profiles.json"),
	}

	cfg := &Config{
		Username:      "testuser",
		ServerURL:     "ws://localhost:8080",
		IsAdmin:       true,
		UseE2E:        true,
		SkipTLSVerify: true,
		Theme:         "modern",
	}

	command := icl.FormatSanitizedLaunchCommand(cfg)

	// Verify command contains expected parts but with sanitized sensitive data
	if !contains(command, "--server ws://localhost:8080") {
		t.Error("Expected command to contain server URL")
	}

	if !contains(command, "--username testuser") {
		t.Error("Expected command to contain username")
	}

	if !contains(command, "--admin") {
		t.Error("Expected command to contain --admin flag")
	}

	if !contains(command, "--admin-key <your-admin-key>") {
		t.Error("Expected command to contain sanitized admin key")
	}

	if !contains(command, "--e2e") {
		t.Error("Expected command to contain --e2e flag")
	}

	if !contains(command, "--keystore-passphrase <your-passphrase>") {
		t.Error("Expected command to contain sanitized keystore passphrase")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
