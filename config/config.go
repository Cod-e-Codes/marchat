package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	// Server settings
	Port     int      `json:"port"`
	AdminKey string   `json:"admin_key"`
	Admins   []string `json:"admins"`

	// TLS settings
	TLSCertFile string `json:"tls_cert_file"`
	TLSKeyFile  string `json:"tls_key_file"`

	// Database settings
	DBPath string `json:"db_path"`

	// Logging
	LogLevel string `json:"log_level"`

	// JWT settings
	JWTSecret string `json:"jwt_secret"`

	// Config directory
	ConfigDir string `json:"config_dir"`

	// Ban history gaps feature
	BanGapsHistory bool `json:"ban_gaps_history"`

	// Plugin settings
	PluginRegistryURL string `json:"plugin_registry_url"`

	// File transfer settings
	MaxFileBytes int64 `json:"max_file_bytes"`

	// E2E encryption settings
	GlobalE2EKey string `json:"global_e2e_key"`
}

// LoadConfig loads configuration from environment variables, .env files, and config files
func LoadConfig(configDir string) (*Config, error) {
	cfg, err := LoadConfigWithoutValidation(configDir)
	if err != nil {
		return nil, err
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// LoadConfigWithoutValidation loads configuration without validation (for interactive setup)
func LoadConfigWithoutValidation(configDir string) (*Config, error) {
	cfg := &Config{}

	// Set config directory - check environment variable first, then parameter, then default
	if envConfigDir := os.Getenv("MARCHAT_CONFIG_DIR"); envConfigDir != "" {
		cfg.ConfigDir = envConfigDir
	} else if configDir != "" {
		cfg.ConfigDir = configDir
	} else {
		cfg.ConfigDir = getDefaultConfigDir()
	}

	// Ensure config directory exists
	if err := ensureConfigDir(cfg.ConfigDir); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load .env file from config directory
	envPath := filepath.Join(cfg.ConfigDir, ".env")
	if err := loadEnvFile(envPath); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	// Load configuration from environment variables
	if err := cfg.loadFromEnv(); err != nil {
		return nil, fmt.Errorf("failed to load environment configuration: %w", err)
	}

	return cfg, nil
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() error {
	// Port configuration
	if portStr := os.Getenv("MARCHAT_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid MARCHAT_PORT: %s", portStr)
		}
		c.Port = port
	} else {
		c.Port = 8080 // Default port
	}

	// Admin key configuration
	if adminKey := os.Getenv("MARCHAT_ADMIN_KEY"); adminKey != "" {
		c.AdminKey = adminKey
	}

	// Admin users configuration
	if usersStr := os.Getenv("MARCHAT_USERS"); usersStr != "" {
		c.Admins = strings.Split(usersStr, ",")
		// Trim whitespace from usernames
		for i, user := range c.Admins {
			c.Admins[i] = strings.TrimSpace(user)
		}
	}

	// Database path configuration
	if dbPath := os.Getenv("MARCHAT_DB_PATH"); dbPath != "" {
		c.DBPath = dbPath
	} else {
		c.DBPath = filepath.Join(c.ConfigDir, "marchat.db")
	}

	// Log level configuration
	if logLevel := os.Getenv("MARCHAT_LOG_LEVEL"); logLevel != "" {
		c.LogLevel = logLevel
	} else {
		c.LogLevel = "info"
	}

	// JWT secret configuration
	if jwtSecret := os.Getenv("MARCHAT_JWT_SECRET"); jwtSecret != "" {
		c.JWTSecret = jwtSecret
	} else {
		jwtSecret, err := generateJWTSecret()
		if err != nil {
			return fmt.Errorf("when generating JWT secret: %w", err)
		}
		c.JWTSecret = jwtSecret
	}

	// TLS configuration
	if tlsCertFile := os.Getenv("MARCHAT_TLS_CERT_FILE"); tlsCertFile != "" {
		c.TLSCertFile = tlsCertFile
	}
	if tlsKeyFile := os.Getenv("MARCHAT_TLS_KEY_FILE"); tlsKeyFile != "" {
		c.TLSKeyFile = tlsKeyFile
	}

	// Ban history gaps configuration
	if banGapsStr := os.Getenv("MARCHAT_BAN_HISTORY_GAPS"); banGapsStr != "" {
		c.BanGapsHistory = strings.ToLower(banGapsStr) == "true"
	} else {
		c.BanGapsHistory = false // Default to false for backward compatibility
	}

	// Plugin registry URL configuration
	if pluginRegistryURL := os.Getenv("MARCHAT_PLUGIN_REGISTRY_URL"); pluginRegistryURL != "" {
		c.PluginRegistryURL = pluginRegistryURL
	} else {
		c.PluginRegistryURL = "https://raw.githubusercontent.com/Cod-e-Codes/marchat-plugins/main/registry.json"
	}

	// Max file size configuration (bytes or MB)
	// Priority: MARCHAT_MAX_FILE_BYTES > MARCHAT_MAX_FILE_MB > default 1MB
	const oneMB int64 = 1024 * 1024
	if bytesStr := os.Getenv("MARCHAT_MAX_FILE_BYTES"); bytesStr != "" {
		val, err := strconv.ParseInt(bytesStr, 10, 64)
		if err != nil || val <= 0 {
			return fmt.Errorf("invalid MARCHAT_MAX_FILE_BYTES: %s", bytesStr)
		}
		c.MaxFileBytes = val
	} else if mbStr := os.Getenv("MARCHAT_MAX_FILE_MB"); mbStr != "" {
		val, err := strconv.ParseInt(mbStr, 10, 64)
		if err != nil || val <= 0 {
			return fmt.Errorf("invalid MARCHAT_MAX_FILE_MB: %s", mbStr)
		}
		c.MaxFileBytes = val * oneMB
	} else {
		c.MaxFileBytes = oneMB
	}

	// Global E2E key configuration
	if globalE2EKey := os.Getenv("MARCHAT_GLOBAL_E2E_KEY"); globalE2EKey != "" {
		c.GlobalE2EKey = globalE2EKey
	}

	return nil
}

// Validate ensures all required configuration is present
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", c.Port)
	}

	if c.AdminKey == "" {
		return fmt.Errorf("MARCHAT_ADMIN_KEY is required")
	}

	if len(c.Admins) == 0 {
		return fmt.Errorf("at least one admin user is required (set MARCHAT_USERS)")
	}

	// Validate admin usernames
	adminSet := make(map[string]struct{})
	for _, user := range c.Admins {
		if user == "" {
			return fmt.Errorf("admin username cannot be empty")
		}
		lu := strings.ToLower(user)
		if _, exists := adminSet[lu]; exists {
			return fmt.Errorf("duplicate admin username (case-insensitive): %s", user)
		}
		adminSet[lu] = struct{}{}
	}

	// Normalize admin usernames to lowercase
	normalizedAdmins := make([]string, 0, len(adminSet))
	for user := range adminSet {
		normalizedAdmins = append(normalizedAdmins, user)
	}
	c.Admins = normalizedAdmins

	return nil
}

// getDefaultConfigDir returns the default configuration directory
func getDefaultConfigDir() string {
	// Check if we're in development mode (running from project root)
	if _, err := os.Stat("go.mod"); err == nil {
		return "./config"
	}

	// Production mode - use XDG config home
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "marchat")
	}

	// Fallback to user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./config"
	}

	return filepath.Join(homeDir, ".config", "marchat")
}

// ensureConfigDir creates the configuration directory if it doesn't exist
func ensureConfigDir(configDir string) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}
	return nil
}

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(envPath string) error {
	// Check if .env file exists
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		// .env file doesn't exist, which is fine
		return nil
	}

	// Load .env file
	if err := godotenv.Load(envPath); err != nil {
		return fmt.Errorf("failed to load .env file %s: %w", envPath, err)
	}

	return nil
}

// GetEnvWithDefault returns an environment variable value or a default
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvIntWithDefault returns an environment variable as int or a default
func GetEnvIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// IsTLSEnabled returns true if both TLS certificate and key files are configured
func (c *Config) IsTLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != ""
}

// GetWebSocketScheme returns the appropriate WebSocket scheme based on TLS configuration
func (c *Config) GetWebSocketScheme() string {
	if c.IsTLSEnabled() {
		return "wss"
	}
	return "ws"
}

func generateJWTSecret() (string, error) {
	jwtSecret := make([]byte, 32)
	_, err := rand.Read(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("when reading random bytes: %w", err)
	}
	return hex.EncodeToString(jwtSecret), nil
}
