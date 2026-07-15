package host

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/plugin/sdk"
)

func TestNewPluginHost(t *testing.T) {
	pluginDir := "/tmp/test-plugins"
	dataDir := "/tmp/test-data"

	host := NewPluginHost(pluginDir, dataDir)

	if host == nil {
		t.Fatal("NewPluginHost returned nil")
		return
	}

	if host.pluginDir != pluginDir {
		t.Errorf("Expected pluginDir %s, got %s", pluginDir, host.pluginDir)
	}

	if host.dataDir != dataDir {
		t.Errorf("Expected dataDir %s, got %s", dataDir, host.dataDir)
	}

	if host.plugins == nil {
		t.Error("plugins map should be initialized")
	}

	if host.messageChan == nil {
		t.Error("messageChan should be initialized")
	}
}

func TestLoadPlugin(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	host := NewPluginHost(pluginDir, dataDir)

	// Test loading non-existent plugin (should return error)
	err := host.LoadPlugin("non-existent-plugin")
	if err == nil {
		t.Error("Expected error when loading non-existent plugin")
	}

	// Test that host is properly initialized
	if host.pluginDir != pluginDir {
		t.Errorf("Expected pluginDir %s, got %s", pluginDir, host.pluginDir)
	}

	if host.dataDir != dataDir {
		t.Errorf("Expected dataDir %s, got %s", dataDir, host.dataDir)
	}
}

func TestLoadPluginInvalidManifest(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	host := NewPluginHost(pluginDir, dataDir)

	// Create a test plugin directory with invalid manifest
	pluginName := "invalid-plugin"
	pluginPath := filepath.Join(pluginDir, pluginName)
	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	// Create an invalid manifest (missing required fields)
	manifestPath := filepath.Join(pluginPath, "plugin.json")
	if err := os.WriteFile(manifestPath, []byte(`{"name": "test"}`), 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	// Test loading the plugin - should fail
	err := host.LoadPlugin(pluginName)
	if err == nil {
		t.Fatal("Expected error when loading plugin with invalid manifest")
	}
}

func TestLoadPluginMissingBinary(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	host := NewPluginHost(pluginDir, dataDir)

	// Create a test plugin directory with valid manifest but no binary
	pluginName := "no-binary-plugin"
	pluginPath := filepath.Join(pluginDir, pluginName)
	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	// Create a valid manifest
	manifest := sdk.PluginManifest{
		Name:        pluginName,
		Version:     "1.0.0",
		Description: "Test plugin",
		Author:      "Test Author",
		License:     "MIT",
	}

	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	manifestPath := filepath.Join(pluginPath, "plugin.json")
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	// Test loading the plugin - should fail due to missing binary
	err = host.LoadPlugin(pluginName)
	if err == nil {
		t.Fatal("Expected error when loading plugin with missing binary")
	}
}

func TestListPlugins(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	host := NewPluginHost(pluginDir, dataDir)

	// Initially should be empty
	plugins := host.ListPlugins()
	if len(plugins) != 0 {
		t.Errorf("Expected empty plugin list, got %d plugins", len(plugins))
	}

	// Create and load a test plugin
	pluginName := "test-plugin"
	pluginPath := filepath.Join(pluginDir, pluginName)
	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	manifest := sdk.PluginManifest{
		Name:        pluginName,
		Version:     "1.0.0",
		Description: "Test plugin",
		Author:      "Test Author",
		License:     "MIT",
	}

	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	manifestPath := filepath.Join(pluginPath, "plugin.json")
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	binaryPath := filepath.Join(pluginPath, pluginName)
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\necho 'test'"), 0755); err != nil {
		t.Fatalf("Failed to create binary: %v", err)
	}

	if err := host.LoadPlugin(pluginName); err != nil {
		t.Fatalf("Failed to load plugin: %v", err)
	}

	// Now should have one plugin
	plugins = host.ListPlugins()
	if len(plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(plugins))
	}

	if _, exists := plugins[pluginName]; !exists {
		t.Errorf("Plugin %s not found in list", pluginName)
	}
}

func TestEnableDisablePlugin(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	host := NewPluginHost(pluginDir, dataDir)

	// Test disabling non-existent plugin (should not panic)
	err := host.DisablePlugin("non-existent-plugin")
	if err == nil {
		t.Error("Expected error when disabling non-existent plugin")
	}

	// Test enabling non-existent plugin (should not panic)
	err = host.EnablePlugin("non-existent-plugin")
	if err == nil {
		t.Error("Expected error when enabling non-existent plugin")
	}

	// Test that host is properly initialized
	if host.pluginDir != pluginDir {
		t.Errorf("Expected pluginDir %s, got %s", pluginDir, host.pluginDir)
	}

	if host.dataDir != dataDir {
		t.Errorf("Expected dataDir %s, got %s", dataDir, host.dataDir)
	}
}

func TestEnableDisableNonExistentPlugin(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	host := NewPluginHost(pluginDir, dataDir)

	// Test enabling non-existent plugin
	err := host.EnablePlugin("non-existent")
	if err == nil {
		t.Fatal("Expected error when enabling non-existent plugin")
	}

	// Test disabling non-existent plugin
	err = host.DisablePlugin("non-existent")
	if err == nil {
		t.Fatal("Expected error when disabling non-existent plugin")
	}
}

func TestSendMessage(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	host := NewPluginHost(pluginDir, dataDir)

	// Test sending message without any plugins loaded
	testMessage := sdk.Message{
		Sender:    "test-user",
		Content:   "Hello plugin!",
		CreatedAt: time.Now(),
	}

	// This should not panic or error (no plugins loaded)
	host.SendMessage(testMessage)

	// Test that message channel is working
	channel := host.GetMessageChannel()
	if channel == nil {
		t.Fatal("Message channel should not be nil")
		return
	}
}

func TestUpdateUserList(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	host := NewPluginHost(pluginDir, dataDir)

	// Test updating user list
	users := []string{"user1", "user2", "user3"}
	host.UpdateUserList(users)

	// Verify user list was updated
	if len(host.userList) != len(users) {
		t.Errorf("Expected %d users, got %d", len(users), len(host.userList))
	}

	for i, user := range users {
		if host.userList[i] != user {
			t.Errorf("Expected user %s at index %d, got %s", user, i, host.userList[i])
		}
	}
}

func TestGetMessageChannel(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	host := NewPluginHost(pluginDir, dataDir)

	// Test getting message channel
	channel := host.GetMessageChannel()
	if channel == nil {
		t.Fatal("Message channel should not be nil")
		return
	}

	// Channel should be readable
	select {
	case <-channel:
		t.Error("Channel should be empty initially")
	default:
		// Expected - channel should be empty
	}
}
