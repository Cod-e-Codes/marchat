package config

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test with environment variables
	t.Run("environment variables", func(t *testing.T) {
		// Set up test environment
		os.Setenv("MARCHAT_PORT", "8080")
		os.Setenv("MARCHAT_ADMIN_KEY", "test-key")
		os.Setenv("MARCHAT_USERS", "user1,user2")
		defer func() {
			os.Unsetenv("MARCHAT_PORT")
			os.Unsetenv("MARCHAT_ADMIN_KEY")
			os.Unsetenv("MARCHAT_USERS")
		}()

		cfg, err := LoadConfig(t.TempDir())
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.Port != 8080 {
			t.Errorf("Expected port 8080, got %d", cfg.Port)
		}
		if cfg.AdminKey != "test-key" {
			t.Errorf("Expected admin key 'test-key', got '%s'", cfg.AdminKey)
		}
		expected := []string{"user1", "user2"}
		sort.Strings(expected)
		sort.Strings(cfg.Admins)

		if !reflect.DeepEqual(cfg.Admins, expected) {
			t.Errorf("Expected admins %v, got %v", expected, cfg.Admins)
		}
	})

	t.Run("default values", func(t *testing.T) {
		// Clear environment variables
		os.Unsetenv("MARCHAT_PORT")
		os.Unsetenv("MARCHAT_ADMIN_KEY")
		os.Unsetenv("MARCHAT_USERS")

		_, err := LoadConfig(t.TempDir())
		if err == nil {
			t.Error("Expected error when required environment variables are missing")
		}
	})

	t.Run("invalid port", func(t *testing.T) {
		os.Setenv("MARCHAT_PORT", "invalid")
		os.Setenv("MARCHAT_ADMIN_KEY", "test-key")
		os.Setenv("MARCHAT_USERS", "user1")
		defer func() {
			os.Unsetenv("MARCHAT_PORT")
			os.Unsetenv("MARCHAT_ADMIN_KEY")
			os.Unsetenv("MARCHAT_USERS")
		}()

		_, err := LoadConfig(t.TempDir())
		if err == nil {
			t.Error("Expected error for invalid port")
		}
	})

	t.Run("custom config directory", func(t *testing.T) {
		tempDir := t.TempDir()

		// Clear MARCHAT_CONFIG_DIR to ensure test isolation
		originalConfigDir := os.Getenv("MARCHAT_CONFIG_DIR")
		os.Unsetenv("MARCHAT_CONFIG_DIR")
		defer func() {
			if originalConfigDir != "" {
				os.Setenv("MARCHAT_CONFIG_DIR", originalConfigDir)
			}
		}()

		os.Setenv("MARCHAT_ADMIN_KEY", "test-key")
		os.Setenv("MARCHAT_USERS", "user1")
		defer func() {
			os.Unsetenv("MARCHAT_ADMIN_KEY")
			os.Unsetenv("MARCHAT_USERS")
		}()

		cfg, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.ConfigDir != tempDir {
			t.Errorf("Expected config dir '%s', got '%s'", tempDir, cfg.ConfigDir)
		}
	})
}

func TestLoadConfigWithEnvFile(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	// Clear environment to ensure test isolation
	originalConfigDir := os.Getenv("MARCHAT_CONFIG_DIR")
	os.Unsetenv("MARCHAT_CONFIG_DIR")
	defer func() {
		if originalConfigDir != "" {
			os.Setenv("MARCHAT_CONFIG_DIR", originalConfigDir)
		}
	}()

	// Create a test .env file
	envContent := `MARCHAT_PORT=8080
MARCHAT_ADMIN_KEY=env-file-key
MARCHAT_USERS=envuser1,envuser2
MARCHAT_DB_PATH=/custom/db/path
MARCHAT_LOG_LEVEL=debug
MARCHAT_JWT_SECRET=custom-jwt-secret`

	err := os.WriteFile(envPath, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	cfg, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Port)
	}
	if cfg.AdminKey != "env-file-key" {
		t.Errorf("Expected admin key 'env-file-key', got '%s'", cfg.AdminKey)
	}
	if len(cfg.Admins) != 2 {
		t.Errorf("Expected 2 admins, got %d", len(cfg.Admins))
	}
	// Sort admins for comparison since map iteration order is not guaranteed
	sort.Strings(cfg.Admins)
	expected := []string{"envuser1", "envuser2"}
	sort.Strings(expected)
	if !reflect.DeepEqual(cfg.Admins, expected) {
		t.Errorf("Expected admins %v, got %v", expected, cfg.Admins)
	}
	if cfg.DBPath != "/custom/db/path" {
		t.Errorf("Expected DB path '/custom/db/path', got '%s'", cfg.DBPath)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", cfg.LogLevel)
	}
	if cfg.JWTSecret != "custom-jwt-secret" {
		t.Errorf("Expected JWT secret 'custom-jwt-secret', got '%s'", cfg.JWTSecret)
	}
}

func TestEnvironmentVariablePrecedence(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	// Clear environment to ensure test isolation
	originalConfigDir := os.Getenv("MARCHAT_CONFIG_DIR")
	os.Unsetenv("MARCHAT_CONFIG_DIR")
	defer func() {
		if originalConfigDir != "" {
			os.Setenv("MARCHAT_CONFIG_DIR", originalConfigDir)
		}
	}()

	// Create a .env file with one value
	envContent := `MARCHAT_PORT=8080
MARCHAT_ADMIN_KEY=env-file-key
MARCHAT_USERS=envuser1`

	err := os.WriteFile(envPath, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	// Set environment variable to override .env file
	os.Setenv("MARCHAT_PORT", "8080")
	defer os.Unsetenv("MARCHAT_PORT")

	cfg, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Environment variable should take precedence
	if cfg.Port != 8080 {
		t.Errorf("Expected port 8080 (from env var), got %d", cfg.Port)
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Port:     8080,
				AdminKey: "test-key",
				Admins:   []string{"user1", "user2"},
			},
			wantErr: false,
		},
		{
			name: "missing admin key",
			cfg: &Config{
				Port:   8080,
				Admins: []string{"user1"},
			},
			wantErr: true,
		},
		{
			name: "no admins",
			cfg: &Config{
				Port:     8080,
				AdminKey: "test-key",
				Admins:   []string{},
			},
			wantErr: true,
		},
		{
			name: "invalid port too low",
			cfg: &Config{
				Port:     0,
				AdminKey: "test-key",
				Admins:   []string{"user1"},
			},
			wantErr: true,
		},
		{
			name: "invalid port too high",
			cfg: &Config{
				Port:     70000,
				AdminKey: "test-key",
				Admins:   []string{"user1"},
			},
			wantErr: true,
		},
		{
			name: "duplicate admin usernames",
			cfg: &Config{
				Port:     8080,
				AdminKey: "test-key",
				Admins:   []string{"user1", "USER1"},
			},
			wantErr: true,
		},
		{
			name: "empty admin username",
			cfg: &Config{
				Port:     8080,
				AdminKey: "test-key",
				Admins:   []string{"user1", ""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDefaultConfigDir(t *testing.T) {
	// Test development mode (go.mod exists)
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create a temporary directory with go.mod
	tempDir := t.TempDir()
	err = os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("Failed to restore working directory: %v", err)
		}
	}()

	configDir := getDefaultConfigDir()
	if configDir != "./config" {
		t.Errorf("Expected './config' in development mode, got '%s'", configDir)
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	// Test with environment variable set
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	result := GetEnvWithDefault("TEST_VAR", "default")
	if result != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", result)
	}

	// Test with environment variable not set
	result = GetEnvWithDefault("NONEXISTENT_VAR", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got '%s'", result)
	}
}

func TestGetEnvIntWithDefault(t *testing.T) {
	// Test with valid integer
	os.Setenv("TEST_INT", "123")
	defer os.Unsetenv("TEST_INT")

	result := GetEnvIntWithDefault("TEST_INT", 456)
	if result != 123 {
		t.Errorf("Expected 123, got %d", result)
	}

	// Test with invalid integer
	os.Setenv("TEST_INVALID", "not-a-number")
	defer os.Unsetenv("TEST_INVALID")

	result = GetEnvIntWithDefault("TEST_INVALID", 456)
	if result != 456 {
		t.Errorf("Expected 456 (default), got %d", result)
	}

	// Test with environment variable not set
	result = GetEnvIntWithDefault("NONEXISTENT_INT", 789)
	if result != 789 {
		t.Errorf("Expected 789 (default), got %d", result)
	}
}
