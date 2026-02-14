package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerConfigUISavesEnv(t *testing.T) {
	// Ensure we use a temp config dir
	tdir := t.TempDir()
	os.Setenv("MARCHAT_CONFIG_DIR", tdir)
	defer os.Unsetenv("MARCHAT_CONFIG_DIR")

	m := NewServerConfigUI()
	// Simulate values
	m.inputs[adminKeyField].SetValue("adminkey123")
	m.inputs[adminUsersField].SetValue("alice,bob")
	m.inputs[portField].SetValue("8123")

	if err := m.validateAndBuildConfig(); err != nil {
		t.Fatalf("validateAndBuildConfig: %v", err)
	}

	envPath := filepath.Join(tdir, ".env")
	if _, err := os.Stat(envPath); err != nil {
		t.Fatalf(".env not written: %v", err)
	}

	// Basic content checks
	b, _ := os.ReadFile(envPath)
	content := string(b)
	for _, needle := range []string{
		"MARCHAT_PORT=8123",
		"MARCHAT_ADMIN_KEY=adminkey123",
		"MARCHAT_USERS=alice,bob",
		"MARCHAT_JWT_SECRET=", // The value is random, so we only check for the presence of the env var
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("missing .env content: %s", needle)
		}
	}
}
