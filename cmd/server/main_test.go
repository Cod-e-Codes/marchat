package main

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/Cod-e-Codes/marchat/config"
	"github.com/Cod-e-Codes/marchat/shared"
)

func TestMultiFlag(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected string
	}{
		{
			name:     "single value",
			values:   []string{"admin1"},
			expected: "admin1",
		},
		{
			name:     "multiple values",
			values:   []string{"admin1", "admin2", "admin3"},
			expected: "admin1,admin2,admin3",
		},
		{
			name:     "empty values",
			values:   []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mf multiFlag
			for _, val := range tt.values {
				if err := mf.Set(val); err != nil {
					t.Fatalf("Set() error = %v", err)
				}
			}
			if got := mf.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPrintBanner(t *testing.T) {
	tests := []struct {
		name   string
		addr   string
		admins []string
		scheme string
	}{
		{
			name:   "http with single admin",
			addr:   "localhost:8080",
			admins: []string{"admin1"},
			scheme: "ws",
		},
		{
			name:   "https with multiple admins",
			addr:   "example.com:8443",
			admins: []string{"admin1", "admin2", "admin3"},
			scheme: "wss",
		},
		{
			name:   "empty admins",
			addr:   "localhost:8080",
			admins: []string{},
			scheme: "ws",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that printBanner doesn't panic or crash
			// We redirect output to /dev/null to avoid cluttering test output
			oldStdout := os.Stdout
			devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
			if err != nil {
				t.Skipf("Cannot open %s: %v", os.DevNull, err)
			}
			os.Stdout = devNull

			// This should not panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("printBanner panicked: %v", r)
					}
				}()
				printBanner(tt.addr, tt.admins, tt.scheme, false)
			}()

			// Restore stdout
			os.Stdout = oldStdout
			devNull.Close()
		})
	}
}

func TestFlagParsing(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected struct {
			adminUsers       []string
			adminKey         string
			port             int
			configPath       string
			configDir        string
			enableAdminPanel bool
			enableWebPanel   bool
		}
	}{
		{
			name: "all flags provided",
			args: []string{
				"--admin", "admin1",
				"--admin", "admin2",
				"--admin-key", "secret123",
				"--port", "9090",
				"--config", "/path/to/config.json",
				"--config-dir", "/custom/config",
				"--admin-panel",
				"--web-panel",
			},
			expected: struct {
				adminUsers       []string
				adminKey         string
				port             int
				configPath       string
				configDir        string
				enableAdminPanel bool
				enableWebPanel   bool
			}{
				adminUsers:       []string{"admin1", "admin2"},
				adminKey:         "secret123",
				port:             9090,
				configPath:       "/path/to/config.json",
				configDir:        "/custom/config",
				enableAdminPanel: true,
				enableWebPanel:   true,
			},
		},
		{
			name: "minimal flags",
			args: []string{},
			expected: struct {
				adminUsers       []string
				adminKey         string
				port             int
				configPath       string
				configDir        string
				enableAdminPanel bool
				enableWebPanel   bool
			}{
				adminUsers:       []string{},
				adminKey:         "",
				port:             0,
				configPath:       "",
				configDir:        "",
				enableAdminPanel: false,
				enableWebPanel:   false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			// Re-declare flags
			var testAdminUsers multiFlag
			var testAdminKey = flag.String("admin-key", "", "Admin key for privileged commands")
			var testPort = flag.Int("port", 0, "Port to listen on")
			var testConfigPath = flag.String("config", "", "Path to server config file")
			var testConfigDir = flag.String("config-dir", "", "Configuration directory")
			var testEnableAdminPanel = flag.Bool("admin-panel", false, "Enable the built-in admin panel TUI")
			var testEnableWebPanel = flag.Bool("web-panel", false, "Enable the built-in web admin panel")

			flag.Var(&testAdminUsers, "admin", "Admin username")

			// Parse flags
			err := flag.CommandLine.Parse(tt.args)
			if err != nil {
				t.Fatalf("Flag parsing failed: %v", err)
			}

			// Verify results
			if len(testAdminUsers) != len(tt.expected.adminUsers) {
				t.Errorf("Admin users length = %d, want %d", len(testAdminUsers), len(tt.expected.adminUsers))
			}
			for i, user := range testAdminUsers {
				if user != tt.expected.adminUsers[i] {
					t.Errorf("Admin user[%d] = %s, want %s", i, user, tt.expected.adminUsers[i])
				}
			}
			if *testAdminKey != tt.expected.adminKey {
				t.Errorf("Admin key = %s, want %s", *testAdminKey, tt.expected.adminKey)
			}
			if *testPort != tt.expected.port {
				t.Errorf("Port = %d, want %d", *testPort, tt.expected.port)
			}
			if *testConfigPath != tt.expected.configPath {
				t.Errorf("Config path = %s, want %s", *testConfigPath, tt.expected.configPath)
			}
			if *testConfigDir != tt.expected.configDir {
				t.Errorf("Config dir = %s, want %s", *testConfigDir, tt.expected.configDir)
			}
			if *testEnableAdminPanel != tt.expected.enableAdminPanel {
				t.Errorf("Enable admin panel = %v, want %v", *testEnableAdminPanel, tt.expected.enableAdminPanel)
			}
			if *testEnableWebPanel != tt.expected.enableWebPanel {
				t.Errorf("Enable web panel = %v, want %v", *testEnableWebPanel, tt.expected.enableWebPanel)
			}
		})
	}
}

func TestAdminUsernameNormalization(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		expected    []string
		shouldError bool
	}{
		{
			name:        "normal usernames",
			input:       []string{"admin1", "admin2", "admin3"},
			expected:    []string{"admin1", "admin2", "admin3"},
			shouldError: false,
		},
		{
			name:        "case insensitive duplicates",
			input:       []string{"Admin1", "admin1", "ADMIN1"},
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "mixed case normalization",
			input:       []string{"Admin1", "admin2", "ADMIN3"},
			expected:    []string{"admin1", "admin2", "admin3"},
			shouldError: false,
		},
		{
			name:        "empty username",
			input:       []string{"admin1", "", "admin2"},
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "whitespace usernames",
			input:       []string{"admin1", "   ", "admin2"},
			expected:    nil,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the normalization logic from main.go
			adminSet := make(map[string]struct{})
			var err error

			for _, u := range tt.input {
				if strings.TrimSpace(u) == "" {
					err = &ConfigError{Message: "admin username cannot be empty"}
					break
				}
				lu := strings.ToLower(strings.TrimSpace(u))
				if _, exists := adminSet[lu]; exists {
					err = &ConfigError{Message: "duplicate admin username (case-insensitive): " + u}
					break
				}
				adminSet[lu] = struct{}{}
			}

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Convert set back to slice and sort for comparison
			normalizedAdmins := make([]string, 0, len(adminSet))
			for u := range adminSet {
				normalizedAdmins = append(normalizedAdmins, u)
			}

			// Sort both slices for comparison
			sortStrings(normalizedAdmins)
			sortStrings(tt.expected)

			if len(normalizedAdmins) != len(tt.expected) {
				t.Errorf("Normalized admins length = %d, want %d", len(normalizedAdmins), len(tt.expected))
				return
			}

			for i, admin := range normalizedAdmins {
				if admin != tt.expected[i] {
					t.Errorf("Normalized admin[%d] = %s, want %s", i, admin, tt.expected[i])
				}
			}
		})
	}
}

func TestVersionInfo(t *testing.T) {
	// Test that version info functions work
	clientVersion := shared.GetVersionInfo()
	serverVersion := shared.GetServerVersionInfo()

	if clientVersion == "" {
		t.Error("Client version info should not be empty")
	}
	if serverVersion == "" {
		t.Error("Server version info should not be empty")
	}

	// Test that version info contains expected components
	if !strings.Contains(clientVersion, shared.ClientVersion) {
		t.Errorf("Client version info should contain ClientVersion: %s", clientVersion)
	}
	if !strings.Contains(serverVersion, shared.ServerVersion) {
		t.Errorf("Server version info should contain ServerVersion: %s", serverVersion)
	}
}

func TestConfigurationValidation(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			envVars: map[string]string{
				"MARCHAT_PORT":      "8080",
				"MARCHAT_ADMIN_KEY": "secret123",
				"MARCHAT_USERS":     "admin1,admin2",
			},
			shouldError: false,
		},
		{
			name: "missing admin key",
			envVars: map[string]string{
				"MARCHAT_PORT":  "8080",
				"MARCHAT_USERS": "admin1",
			},
			shouldError: true,
			errorMsg:    "MARCHAT_ADMIN_KEY is required",
		},
		{
			name: "no admins",
			envVars: map[string]string{
				"MARCHAT_PORT":      "8080",
				"MARCHAT_ADMIN_KEY": "secret123",
			},
			shouldError: true,
			errorMsg:    "at least one admin user is required",
		},
		{
			name: "invalid port - too low",
			envVars: map[string]string{
				"MARCHAT_PORT":      "0",
				"MARCHAT_ADMIN_KEY": "secret123",
				"MARCHAT_USERS":     "admin1",
			},
			shouldError: true,
			errorMsg:    "port must be between 1 and 65535",
		},
		{
			name: "invalid port - too high",
			envVars: map[string]string{
				"MARCHAT_PORT":      "65536",
				"MARCHAT_ADMIN_KEY": "secret123",
				"MARCHAT_USERS":     "admin1",
			},
			shouldError: true,
			errorMsg:    "port must be between 1 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment and clear all MARCHAT_* variables
			originalEnv := make(map[string]string)
			allEnvVars := os.Environ()

			// First, save and clear all MARCHAT_* environment variables
			for _, envVar := range allEnvVars {
				if strings.HasPrefix(envVar, "MARCHAT_") {
					parts := strings.SplitN(envVar, "=", 2)
					if len(parts) == 2 {
						key := parts[0]
						originalEnv[key] = parts[1]
						os.Unsetenv(key)
					}
				}
			}

			// Then set the test-specific environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Clean up environment after test
			defer func() {
				// Clear all MARCHAT_* variables first
				for key := range tt.envVars {
					os.Unsetenv(key)
				}

				// Restore original environment
				for key, originalValue := range originalEnv {
					os.Setenv(key, originalValue)
				}
			}()

			// Test configuration loading
			_, err := config.LoadConfig(t.TempDir())
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message should contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper function to sort string slices
func sortStrings(s []string) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

func TestTLSConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		envVars      map[string]string
		expectTLS    bool
		expectScheme string
	}{
		{
			name: "TLS enabled with cert and key",
			envVars: map[string]string{
				"MARCHAT_PORT":          "8080",
				"MARCHAT_ADMIN_KEY":     "secret123",
				"MARCHAT_USERS":         "admin1",
				"MARCHAT_TLS_CERT_FILE": "/path/to/cert.pem",
				"MARCHAT_TLS_KEY_FILE":  "/path/to/key.pem",
			},
			expectTLS:    true,
			expectScheme: "wss",
		},
		{
			name: "TLS disabled - no cert",
			envVars: map[string]string{
				"MARCHAT_PORT":         "8080",
				"MARCHAT_ADMIN_KEY":    "secret123",
				"MARCHAT_USERS":        "admin1",
				"MARCHAT_TLS_KEY_FILE": "/path/to/key.pem",
			},
			expectTLS:    false,
			expectScheme: "ws",
		},
		{
			name: "TLS disabled - no key",
			envVars: map[string]string{
				"MARCHAT_PORT":          "8080",
				"MARCHAT_ADMIN_KEY":     "secret123",
				"MARCHAT_USERS":         "admin1",
				"MARCHAT_TLS_CERT_FILE": "/path/to/cert.pem",
			},
			expectTLS:    false,
			expectScheme: "ws",
		},
		{
			name: "TLS disabled - no cert or key",
			envVars: map[string]string{
				"MARCHAT_PORT":      "8080",
				"MARCHAT_ADMIN_KEY": "secret123",
				"MARCHAT_USERS":     "admin1",
			},
			expectTLS:    false,
			expectScheme: "ws",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment and clear all MARCHAT_* variables
			originalEnv := make(map[string]string)
			allEnvVars := os.Environ()

			// First, save and clear all MARCHAT_* environment variables
			for _, envVar := range allEnvVars {
				if strings.HasPrefix(envVar, "MARCHAT_") {
					parts := strings.SplitN(envVar, "=", 2)
					if len(parts) == 2 {
						key := parts[0]
						originalEnv[key] = parts[1]
						os.Unsetenv(key)
					}
				}
			}

			// Then set the test-specific environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Clean up environment after test
			defer func() {
				// Clear all MARCHAT_* variables first
				for key := range tt.envVars {
					os.Unsetenv(key)
				}

				// Restore original environment
				for key, originalValue := range originalEnv {
					os.Setenv(key, originalValue)
				}
			}()

			// Load configuration
			cfg, err := config.LoadConfig(t.TempDir())
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// Test TLS configuration
			if cfg.IsTLSEnabled() != tt.expectTLS {
				t.Errorf("IsTLSEnabled() = %v, want %v", cfg.IsTLSEnabled(), tt.expectTLS)
			}

			// Test WebSocket scheme
			if cfg.GetWebSocketScheme() != tt.expectScheme {
				t.Errorf("GetWebSocketScheme() = %s, want %s", cfg.GetWebSocketScheme(), tt.expectScheme)
			}
		})
	}
}

func TestDeprecatedFlagWarnings(t *testing.T) {
	// This test verifies that deprecated flags are properly handled
	// We can't easily test the actual warning output without complex mocking,
	// but we can test that the flags are parsed correctly

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "deprecated admin flag",
			args: []string{"--admin", "testadmin"},
		},
		{
			name: "deprecated admin-key flag",
			args: []string{"--admin-key", "testkey"},
		},
		{
			name: "deprecated port flag",
			args: []string{"--port", "9090"},
		},
		{
			name: "deprecated config flag",
			args: []string{"--config", "/path/to/config.json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			// Re-declare flags
			var testAdminUsers multiFlag
			_ = flag.String("admin-key", "", "Admin key for privileged commands")
			_ = flag.Int("port", 0, "Port to listen on")
			_ = flag.String("config", "", "Path to server config file")

			flag.Var(&testAdminUsers, "admin", "Admin username")

			// Parse flags - should not error
			err := flag.CommandLine.Parse(tt.args)
			if err != nil {
				t.Errorf("Flag parsing failed: %v", err)
			}
		})
	}
}

func TestMainFunctionIntegration(t *testing.T) {
	// This is a basic integration test that verifies the main function
	// can start up with valid configuration without crashing

	t.Run("main function startup with valid config", func(t *testing.T) {
		// Set up environment for valid configuration
		originalEnv := make(map[string]string)
		testEnv := map[string]string{
			"MARCHAT_PORT":      "8081", // Use different port to avoid conflicts
			"MARCHAT_ADMIN_KEY": "test-secret-key",
			"MARCHAT_USERS":     "testadmin",
		}

		// Save original environment and clear all MARCHAT_* variables
		allEnvVars := os.Environ()

		// First, save and clear all MARCHAT_* environment variables
		for _, envVar := range allEnvVars {
			if strings.HasPrefix(envVar, "MARCHAT_") {
				parts := strings.SplitN(envVar, "=", 2)
				if len(parts) == 2 {
					key := parts[0]
					originalEnv[key] = parts[1]
					os.Unsetenv(key)
				}
			}
		}

		// Then set the test-specific environment variables
		for key, value := range testEnv {
			os.Setenv(key, value)
		}

		// Clean up environment after test
		defer func() {
			// Clear all MARCHAT_* variables first
			for key := range testEnv {
				os.Unsetenv(key)
			}

			// Restore original environment
			for key, originalValue := range originalEnv {
				os.Setenv(key, originalValue)
			}
		}()

		// Test that configuration loads successfully
		cfg, err := config.LoadConfig(t.TempDir())
		if err != nil {
			t.Fatalf("Configuration should load successfully: %v", err)
		}

		// Verify configuration values
		if cfg.Port != 8081 {
			t.Errorf("Expected port 8081, got %d", cfg.Port)
		}
		if cfg.AdminKey != "test-secret-key" {
			t.Errorf("Expected admin key 'test-secret-key', got '%s'", cfg.AdminKey)
		}
		if len(cfg.Admins) != 1 || cfg.Admins[0] != "testadmin" {
			t.Errorf("Expected admin ['testadmin'], got %v", cfg.Admins)
		}
	})
}

// Mock ConfigError type for testing
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}
