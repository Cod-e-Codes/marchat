package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Deprecated: Config is the legacy JSON-based server configuration.
// The main startup path uses config.Config from the config package instead.
// This type and its loaders are retained only for backward-compatible JSON file loading
// and may be removed in a future release.
type Config struct {
	Port     int      `json:"port"`
	Admins   []string `json:"admins"`
	AdminKey string   `json:"admin_key"`
}

// Deprecated: LoadConfig loads configuration from a JSON file.
// Use config.LoadConfig from the config package for the primary startup path.
func LoadConfig(path string) (Config, error) {
	var cfg Config
	f, err := os.Open(path)
	if err != nil {
		return cfg, fmt.Errorf("could not open config file: %w", err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("could not decode config: %w", err)
	}
	return cfg, nil
}

// Deprecated: LoadConfigFromDir loads configuration from a directory, checking for JSON config files.
// Use config.LoadConfig from the config package for the primary startup path.
func LoadConfigFromDir(configDir string) (Config, error) {
	var cfg Config

	// Check for server_config.json in the config directory
	configPath := filepath.Join(configDir, "server_config.json")
	if _, err := os.Stat(configPath); err == nil {
		return LoadConfig(configPath)
	}

	// Check for server_config.json in the current directory (backward compatibility)
	if _, err := os.Stat("server_config.json"); err == nil {
		return LoadConfig("server_config.json")
	}

	// Return empty config if no JSON config file is found
	// This allows the environment-based config to take precedence
	return cfg, nil
}
