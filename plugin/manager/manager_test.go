package manager

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/plugin/sdk"
)

func TestValidatePluginName(t *testing.T) {
	t.Parallel()

	valid := []string{"echo", "plugin_1", "my-plugin"}
	for _, name := range valid {
		if err := validatePluginName(name); err != nil {
			t.Fatalf("expected valid name %q, got error: %v", name, err)
		}
	}

	invalid := []string{
		"",
		"../escape",
		"/absolute",
		"UpperCase",
		"with.dot",
		strings.Repeat("a", 65),
	}
	for _, name := range invalid {
		if err := validatePluginName(name); err == nil {
			t.Fatalf("expected invalid name %q to return error", name)
		}
	}
}

func TestLoadPluginStateFallbacks(t *testing.T) {
	t.Parallel()

	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")

	// Missing file should return empty map.
	state := manager.loadPluginState()
	if state == nil || state.Enabled == nil {
		t.Fatalf("expected default state with initialized map")
	}
	if len(state.Enabled) != 0 {
		t.Fatalf("expected no enabled plugins, got %d", len(state.Enabled))
	}

	// Corrupted JSON should also return default state.
	if err := os.WriteFile(manager.stateFile, []byte("{not-json"), 0644); err != nil {
		t.Fatalf("failed to write corrupted state file: %v", err)
	}
	state = manager.loadPluginState()
	if state == nil || state.Enabled == nil || len(state.Enabled) != 0 {
		t.Fatalf("expected default state for corrupted json")
	}

	// Valid JSON without enabled map should initialize map.
	if err := os.WriteFile(manager.stateFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write empty state file: %v", err)
	}
	state = manager.loadPluginState()
	if state.Enabled == nil {
		t.Fatalf("expected enabled map to be initialized")
	}
}

func TestNewPluginManager(t *testing.T) {
	pluginDir := "/tmp/test-plugins"
	dataDir := "/tmp/test-data"
	registryURL := "https://example.com/registry.json"

	manager := NewPluginManager(pluginDir, dataDir, registryURL)

	if manager == nil {
		t.Fatal("NewPluginManager returned nil")
	}

	if manager.pluginDir != pluginDir {
		t.Errorf("Expected pluginDir %s, got %s", pluginDir, manager.pluginDir)
	}

	if manager.dataDir != dataDir {
		t.Errorf("Expected dataDir %s, got %s", dataDir, manager.dataDir)
	}

	if manager.registryURL != registryURL {
		t.Errorf("Expected registryURL %s, got %s", registryURL, manager.registryURL)
	}

	if manager.host == nil {
		t.Error("Plugin host should be initialized")
	}

	if manager.store == nil {
		t.Error("Plugin store should be initialized")
	}
}

func TestInstallPluginWithLocalFile(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	// Create a test plugin binary
	pluginName := "test-plugin"
	var pluginBinary []byte
	if runtime.GOOS == "windows" {
		// Create a simple batch file for Windows
		pluginBinary = []byte("@echo off\necho test plugin binary\nexit 0\n")
	} else {
		// Create a bash script for Unix-like systems
		pluginBinary = []byte("#!/bin/bash\necho 'test plugin binary'\nexit 0\n")
	}

	// Create a proper ZIP file for the plugin
	var zipData []byte
	{
		var buf bytes.Buffer
		zipWriter := zip.NewWriter(&buf)

		// Add the plugin binary to the ZIP
		// The host expects the binary to be named exactly like the plugin
		writer, err := zipWriter.Create(pluginName)
		if err != nil {
			t.Fatalf("Failed to create ZIP entry: %v", err)
		}
		if _, err := writer.Write(pluginBinary); err != nil {
			t.Fatalf("Failed to write to ZIP: %v", err)
		}

		// Add a plugin manifest
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

		manifestWriter, err := zipWriter.Create("plugin.json")
		if err != nil {
			t.Fatalf("Failed to create manifest ZIP entry: %v", err)
		}
		if _, err := manifestWriter.Write(manifestData); err != nil {
			t.Fatalf("Failed to write manifest to ZIP: %v", err)
		}

		zipWriter.Close()
		zipData = buf.Bytes()
	}

	// Create a temporary file for the plugin ZIP
	tempFile, err := os.CreateTemp("", "test-plugin-*.zip")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write the ZIP data to the temp file
	if _, err := tempFile.Write(zipData); err != nil {
		t.Fatalf("Failed to write plugin ZIP: %v", err)
	}
	tempFile.Close()

	// Create a mock registry with local file URL
	registry := []map[string]interface{}{
		{
			"name":         pluginName,
			"version":      "1.0.0",
			"description":  "Test plugin",
			"author":       "Test Author",
			"license":      "MIT",
			"download_url": "file://" + tempFile.Name(),
			"category":     "test",
		},
	}

	// Create a temporary registry file
	registryFile := filepath.Join(t.TempDir(), "registry.json")
	registryData, err := json.Marshal(registry)
	if err != nil {
		t.Fatalf("Failed to marshal registry: %v", err)
	}

	if err := os.WriteFile(registryFile, registryData, 0644); err != nil {
		t.Fatalf("Failed to write registry file: %v", err)
	}

	// Create plugin manager with local registry
	absPath, err := filepath.Abs(registryFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	manager := NewPluginManager(pluginDir, dataDir, "file://"+absPath)

	// Load the registry
	store := manager.GetStore()
	if err := store.LoadFromCache(); err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Manually set the plugins in the store for testing
	_ = store.Refresh()

	// Test installing the plugin (but skip the start part to avoid hanging)
	// We'll test the store resolution instead
	storePlugin := store.ResolvePlugin(pluginName, "", "")
	if storePlugin == nil {
		t.Fatal("Plugin not found in store")
	}

	if storePlugin.Name != pluginName {
		t.Errorf("Expected plugin name %s, got %s", pluginName, storePlugin.Name)
	}

	// Test that the plugin can be resolved from the store
	// (We don't actually install it to avoid hanging)
	if storePlugin.DownloadURL != "file://"+tempFile.Name() {
		t.Errorf("Expected download URL %s, got %s", "file://"+tempFile.Name(), storePlugin.DownloadURL)
	}
}

func TestInstallPluginWithHTTP(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	// Create a test plugin binary
	var pluginBinary []byte
	if runtime.GOOS == "windows" {
		// Create a simple batch file for Windows
		pluginBinary = []byte("@echo off\necho test plugin binary\nexit 0\n")
	} else {
		// Create a bash script for Unix-like systems
		pluginBinary = []byte("#!/bin/bash\necho 'test plugin binary'\nexit 0\n")
	}

	// Create a proper ZIP file for the plugin
	var zipData []byte
	{
		var buf bytes.Buffer
		zipWriter := zip.NewWriter(&buf)

		// Add the plugin binary to the ZIP
		// The host expects the binary to be named exactly like the plugin
		writer, err := zipWriter.Create("http-plugin")
		if err != nil {
			t.Fatalf("Failed to create ZIP entry: %v", err)
		}
		if _, err := writer.Write(pluginBinary); err != nil {
			t.Fatalf("Failed to write to ZIP: %v", err)
		}

		// Add a plugin manifest
		manifest := sdk.PluginManifest{
			Name:        "http-plugin",
			Version:     "1.0.0",
			Description: "HTTP test plugin",
			Author:      "Test Author",
			License:     "MIT",
		}

		manifestData, err := json.Marshal(manifest)
		if err != nil {
			t.Fatalf("Failed to marshal manifest: %v", err)
		}

		manifestWriter, err := zipWriter.Create("plugin.json")
		if err != nil {
			t.Fatalf("Failed to create manifest ZIP entry: %v", err)
		}
		if _, err := manifestWriter.Write(manifestData); err != nil {
			t.Fatalf("Failed to write manifest to ZIP: %v", err)
		}

		zipWriter.Close()
		zipData = buf.Bytes()
	}

	// Create a mock HTTP server
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/registry.json":
			// Return registry
			registry := []map[string]interface{}{
				{
					"name":         "http-plugin",
					"version":      "1.0.0",
					"description":  "HTTP test plugin",
					"author":       "Test Author",
					"license":      "MIT",
					"download_url": serverURL + "/plugin.zip",
					"category":     "test",
				},
			}

			registryData, _ := json.Marshal(registry)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(registryData)
		case "/plugin.zip":
			// Return plugin ZIP file
			_, _ = w.Write(zipData)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	// Create plugin manager with HTTP registry
	manager := NewPluginManager(pluginDir, dataDir, server.URL+"/registry.json")

	// Refresh the store to load the registry
	if err := manager.RefreshStore(); err != nil {
		t.Fatalf("Failed to refresh store: %v", err)
	}

	// Test plugin store resolution (without installation to avoid hanging)
	store := manager.GetStore()
	storePlugin := store.ResolvePlugin("http-plugin", "", "")
	if storePlugin == nil {
		t.Fatal("Plugin not found in store")
	}

	if storePlugin.Name != "http-plugin" {
		t.Errorf("Expected plugin name 'http-plugin', got %s", storePlugin.Name)
	}
}

func TestUninstallPlugin(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	// Create a test plugin
	pluginName := "test-plugin"
	pluginPath := filepath.Join(pluginDir, pluginName)
	dataPath := filepath.Join(dataDir, pluginName)

	// Create plugin directory and files
	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	if err := os.MkdirAll(dataPath, 0755); err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}

	// Create plugin binary
	binaryPath := filepath.Join(pluginPath, pluginName)
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\necho 'test'"), 0755); err != nil {
		t.Fatalf("Failed to create binary: %v", err)
	}

	// Create plugin manifest
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

	// Create plugin manager
	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")

	// Test uninstalling non-existent plugin (should return error)
	err = manager.UninstallPlugin("non-existent-plugin")
	if err == nil {
		t.Error("Expected error when uninstalling non-existent plugin")
	}

	// Test that manager is properly initialized
	if manager.GetStore() == nil {
		t.Error("Store should be initialized")
	}
}

func TestEnableDisablePlugin(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	// Create plugin manager
	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")

	// Test disabling non-existent plugin (should not panic)
	err := manager.DisablePlugin("non-existent-plugin")
	if err == nil {
		t.Error("Expected error when disabling non-existent plugin")
	}

	// Test enabling non-existent plugin (should not panic)
	err = manager.EnablePlugin("non-existent-plugin")
	if err == nil {
		t.Error("Expected error when enabling non-existent plugin")
	}

	// Test that manager is properly initialized
	if manager.GetStore() == nil {
		t.Error("Store should be initialized")
	}
}

func TestListPlugins(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	// Create plugin manager
	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")

	// Initially should be empty
	plugins := manager.ListPlugins()
	if len(plugins) != 0 {
		t.Errorf("Expected empty plugin list, got %d plugins", len(plugins))
	}

	// Create and load a test plugin
	pluginName := "test-plugin"
	pluginPath := filepath.Join(pluginDir, pluginName)

	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	// Create plugin binary
	binaryPath := filepath.Join(pluginPath, pluginName)
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\necho 'test'"), 0755); err != nil {
		t.Fatalf("Failed to create binary: %v", err)
	}

	// Create plugin manifest
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

	if err := manager.host.LoadPlugin(pluginName); err != nil {
		t.Fatalf("Failed to load plugin: %v", err)
	}

	// Now should have one plugin
	plugins = manager.ListPlugins()
	if len(plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(plugins))
	}

	if _, exists := plugins[pluginName]; !exists {
		t.Errorf("Plugin %s not found in list", pluginName)
	}
}

func TestSendMessage(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	// Create plugin manager
	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")

	// Create a test message
	message := sdk.Message{
		Sender:    "test-user",
		Content:   "Hello plugin!",
		CreatedAt: time.Now(),
	}

	// Test sending message - should not panic
	manager.SendMessage(message)
}

func TestUpdateUserList(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	// Create plugin manager
	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")

	// Test updating user list
	users := []string{"user1", "user2", "user3"}
	manager.UpdateUserList(users)

	// This should not panic or error
}

func TestGetMessageChannel(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	// Create plugin manager
	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")

	// Test getting message channel
	channel := manager.GetMessageChannel()
	if channel == nil {
		t.Fatal("Message channel should not be nil")
	}
}

func TestRefreshStore(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		registry := []map[string]interface{}{
			{
				"name":         "test-plugin",
				"version":      "1.0.0",
				"description":  "Test plugin",
				"author":       "Test Author",
				"license":      "MIT",
				"download_url": "https://example.com/plugin.zip",
			},
		}

		registryData, _ := json.Marshal(registry)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(registryData)
	}))
	defer server.Close()

	// Create plugin manager
	manager := NewPluginManager(pluginDir, dataDir, server.URL+"/registry.json")

	// Test refreshing store
	err := manager.RefreshStore()
	if err != nil {
		t.Fatalf("Failed to refresh store: %v", err)
	}

	// Verify store was refreshed
	store := manager.GetStore()
	if store == nil {
		t.Fatal("Store should not be nil")
	}
}

func TestLoadStoreFromCache(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	// Create plugin manager
	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")

	// Test loading from cache (should not error even if cache doesn't exist)
	err := manager.LoadStoreFromCache()
	if err != nil {
		t.Fatalf("Failed to load from cache: %v", err)
	}
}

func TestGetPluginCommands(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	// Create plugin manager
	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")

	// Initially should be empty
	commands := manager.GetPluginCommands()
	if len(commands) != 0 {
		t.Errorf("Expected empty commands, got %d", len(commands))
	}

	// Create and load a test plugin with commands
	pluginName := "test-plugin"
	pluginPath := filepath.Join(pluginDir, pluginName)

	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	// Create plugin binary
	binaryPath := filepath.Join(pluginPath, pluginName)
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\necho 'test'"), 0755); err != nil {
		t.Fatalf("Failed to create binary: %v", err)
	}

	// Create plugin manifest with commands
	manifest := sdk.PluginManifest{
		Name:        pluginName,
		Version:     "1.0.0",
		Description: "Test plugin",
		Author:      "Test Author",
		License:     "MIT",
		Commands: []sdk.PluginCommand{
			{
				Name:        "test",
				Description: "Test command",
				Usage:       ":test",
				AdminOnly:   false,
			},
			{
				Name:        "admin-test",
				Description: "Admin test command",
				Usage:       ":admin-test",
				AdminOnly:   true,
			},
		},
	}

	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	manifestPath := filepath.Join(pluginPath, "plugin.json")
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	if err := manager.host.LoadPlugin(pluginName); err != nil {
		t.Fatalf("Failed to load plugin: %v", err)
	}

	// Now should have commands
	commands = manager.GetPluginCommands()
	if len(commands) != 1 {
		t.Errorf("Expected 1 plugin with commands, got %d", len(commands))
	}

	pluginCommands, exists := commands[pluginName]
	if !exists {
		t.Fatal("Plugin commands not found")
	}

	if len(pluginCommands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(pluginCommands))
	}
}

func TestGetPluginManifest(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	// Create plugin manager
	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")

	// Test getting manifest for non-existent plugin
	manifest := manager.GetPluginManifest("non-existent")
	if manifest != nil {
		t.Fatal("Expected nil manifest for non-existent plugin")
	}

	// Create and load a test plugin
	pluginName := "test-plugin"
	pluginPath := filepath.Join(pluginDir, pluginName)

	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	// Create plugin binary
	binaryPath := filepath.Join(pluginPath, pluginName)
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\necho 'test'"), 0755); err != nil {
		t.Fatalf("Failed to create binary: %v", err)
	}

	// Create plugin manifest
	expectedManifest := sdk.PluginManifest{
		Name:        pluginName,
		Version:     "1.0.0",
		Description: "Test plugin",
		Author:      "Test Author",
		License:     "MIT",
	}

	manifestData, err := json.Marshal(expectedManifest)
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	manifestPath := filepath.Join(pluginPath, "plugin.json")
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	if err := manager.host.LoadPlugin(pluginName); err != nil {
		t.Fatalf("Failed to load plugin: %v", err)
	}

	// Test getting manifest for existing plugin
	manifest = manager.GetPluginManifest(pluginName)
	if manifest == nil {
		t.Fatal("Expected manifest for existing plugin")
	}

	if manifest.Name != pluginName {
		t.Errorf("Expected manifest name %s, got %s", pluginName, manifest.Name)
	}
}
