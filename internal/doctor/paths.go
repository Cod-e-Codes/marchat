package doctor

import (
	"os"
	"path/filepath"
)

// ResolveServerConfigDir returns the server configuration directory using the same
// rules as cmd/server: MARCHAT_CONFIG_DIR, --config-dir flag, then dev (./config) or
// ~/.config/marchat.
func ResolveServerConfigDir(configDirFlag string) string {
	if envConfigDir := os.Getenv("MARCHAT_CONFIG_DIR"); envConfigDir != "" {
		return envConfigDir
	}
	if configDirFlag != "" {
		return configDirFlag
	}
	if _, err := os.Stat("go.mod"); err == nil {
		return "./config"
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./config"
	}
	return filepath.Join(homeDir, ".config", "marchat")
}
