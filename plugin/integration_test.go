package plugin

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/plugin/host"
	"github.com/Cod-e-Codes/marchat/plugin/manager"
	"github.com/Cod-e-Codes/marchat/plugin/sdk"
	"github.com/Cod-e-Codes/marchat/plugin/store"
)

func TestPluginSystemIntegration(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	// Create a test plugin binary
	pluginName := "integration-test-plugin"

	// Create a simple executable that works cross-platform
	// We'll create a simple script that just exits immediately
	var pluginBinary []byte
	if runtime.GOOS == "windows" {
		// Create a simple batch file for Windows
		pluginBinary = []byte("@echo off\necho integration test plugin\nexit 0\n")
	} else {
		// Create a bash script for Unix-like systems
		pluginBinary = []byte("#!/bin/bash\necho 'integration test plugin'\nexit 0\n")
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
			Description: "Integration test plugin",
			Author:      "Test Author",
			License:     "MIT",
			Commands: []sdk.PluginCommand{
				{
					Name:        "test",
					Description: "Test command",
					Usage:       ":test",
					AdminOnly:   false,
				},
			},
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

	// Create a mock HTTP server for the registry
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/registry.json":
			// Return registry
			registry := []map[string]interface{}{
				{
					"name":         pluginName,
					"version":      "1.0.0",
					"description":  "Integration test plugin",
					"author":       "Test Author",
					"license":      "MIT",
					"download_url": serverURL + "/plugin.zip",
					"category":     "test",
					"commands": []map[string]interface{}{
						{
							"name":        "test",
							"description": "Test command",
							"usage":       ":test",
							"admin_only":  false,
						},
					},
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

	// Create plugin manager
	manager := manager.NewPluginManager(pluginDir, dataDir, server.URL+"/registry.json")

	// Test 1: Refresh store
	err := manager.RefreshStore()
	if err != nil {
		t.Fatalf("Failed to refresh store: %v", err)
	}

	// Test 2: Test plugin store resolution (without installation to avoid hanging)
	store := manager.GetStore()
	storePlugin := store.ResolvePlugin(pluginName, "", "")
	if storePlugin == nil {
		t.Fatal("Plugin not found in store")
		return
	}

	if storePlugin.Name != pluginName {
		t.Errorf("Expected plugin name %s, got %s", pluginName, storePlugin.Name)
	}

	// Test 3: Test plugin store operations
	plugins := store.GetPlugins()
	if len(plugins) != 1 {
		t.Errorf("Expected 1 plugin in store, got %d", len(plugins))
	}

	// Test 4: Test plugin filtering
	filtered := store.FilterPlugins("test", "", nil)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 filtered plugin, got %d", len(filtered))
	}

	// Test 5: Test plugin categories
	categories := store.GetCategories()
	if len(categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(categories))
	}

	// Test 6: Test plugin tags
	tags := store.GetTags()
	if len(tags) != 0 {
		t.Errorf("Expected 0 tags (no tags in test plugin), got %d", len(tags))
	}

	// Test 7: Test manager operations that don't require running plugins
	manager.SendMessage(sdk.Message{
		Sender:    "test-user",
		Content:   "Hello plugin!",
		CreatedAt: time.Now(),
	}) // Should not panic

	// Test 8: Update user list
	users := []string{"user1", "user2", "user3"}
	manager.UpdateUserList(users)

	// Test 9: Get message channel
	channel := manager.GetMessageChannel()
	if channel == nil {
		t.Fatal("Message channel should not be nil")
		return
	}
}

func TestPluginHostIntegration(t *testing.T) {
	// Create temporary directories
	pluginDir := t.TempDir()
	dataDir := t.TempDir()

	// Create plugin host
	host := host.NewPluginHost(pluginDir, dataDir)

	// Test 1: Send message (without loading plugins)
	testMessage := sdk.Message{
		Sender:    "test-user",
		Content:   "Hello host!",
		CreatedAt: time.Now(),
	}
	host.SendMessage(testMessage) // Should not panic

	// Test 2: Update user list
	users := []string{"user1", "user2"}
	host.UpdateUserList(users)

	// Test 3: Get message channel
	channel := host.GetMessageChannel()
	if channel == nil {
		t.Fatal("Message channel should not be nil")
		return
	}

	// Test 4: List plugins (should be empty initially)
	plugins := host.ListPlugins()
	if len(plugins) != 0 {
		t.Errorf("Expected 0 plugins initially, got %d", len(plugins))
	}
}

func TestPluginStoreIntegration(t *testing.T) {
	// Create temporary directories
	cacheDir := t.TempDir()

	// Create a test registry file
	registryFile := filepath.Join(t.TempDir(), "registry.json")

	// Create test plugins
	plugins := []store.StorePlugin{
		{
			Name:        "store-test-1",
			Version:     "1.0.0",
			Description: "Store test plugin 1",
			Author:      "Test Author",
			License:     "MIT",
			DownloadURL: "https://example.com/plugin1.zip",
			Category:    "utility",
			Tags:        []string{"test", "utility"},
			GoOS:        "linux",
			GoArch:      "amd64",
		},
		{
			Name:        "store-test-1",
			Version:     "1.0.0",
			Description: "Store test plugin 1 (Windows)",
			Author:      "Test Author",
			License:     "MIT",
			DownloadURL: "https://example.com/plugin1-windows.zip",
			Category:    "utility",
			Tags:        []string{"test", "utility"},
			GoOS:        "windows",
			GoArch:      "amd64",
		},
		{
			Name:        "store-test-2",
			Version:     "2.0.0",
			Description: "Store test plugin 2",
			Author:      "Test Author 2",
			License:     "Apache-2.0",
			DownloadURL: "https://example.com/plugin2.zip",
			Category:    "fun",
			Tags:        []string{"test", "fun"},
		},
	}

	// Write registry file
	registryData, err := json.MarshalIndent(plugins, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal registry: %v", err)
	}

	if err := os.WriteFile(registryFile, registryData, 0644); err != nil {
		t.Fatalf("Failed to write registry file: %v", err)
	}

	// Create store
	absPath, err := filepath.Abs(registryFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	store := store.NewStore("file://"+absPath, cacheDir)

	// Test 1: Refresh store
	err = store.Refresh()
	if err != nil {
		t.Fatalf("Failed to refresh store: %v", err)
	}

	// Test 2: Get all plugins
	allPlugins := store.GetPlugins()
	if len(allPlugins) != 3 {
		t.Errorf("Expected 3 plugins, got %d", len(allPlugins))
	}

	// Test 3: Resolve plugin with platform preference
	plugin := store.ResolvePlugin("store-test-1", "linux", "amd64")
	if plugin == nil {
		t.Fatal("Expected to find linux/amd64 plugin")
		return
	}
	if plugin.GoOS != "linux" || plugin.GoArch != "amd64" {
		t.Errorf("Expected linux/amd64 plugin, got %s/%s", plugin.GoOS, plugin.GoArch)
	}

	// Test 4: Get plugins preferred for platform
	preferred := store.GetPluginsPreferredForPlatform("linux", "amd64")
	if len(preferred) != 2 {
		t.Errorf("Expected 2 preferred plugins, got %d", len(preferred))
	}

	// Test 5: Filter plugins by category
	filtered := store.FilterPlugins("utility", "", nil)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 utility plugins, got %d", len(filtered))
	}

	// Test 6: Filter plugins by search term
	filtered = store.FilterPlugins("", "test", nil)
	if len(filtered) != 3 {
		t.Errorf("Expected 3 plugins matching 'test', got %d", len(filtered))
	}

	// Test 7: Filter plugins by tags
	filtered = store.FilterPlugins("", "", []string{"utility"})
	if len(filtered) != 2 {
		t.Errorf("Expected 2 plugins with 'utility' tag, got %d", len(filtered))
	}

	// Test 8: Get categories
	categories := store.GetCategories()
	if len(categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(categories))
	}

	// Test 9: Get tags
	tags := store.GetTags()
	if len(tags) != 3 {
		t.Errorf("Expected 3 unique tags, got %d", len(tags))
	}

	// Test 10: Update installed status
	installedPlugins := map[string]bool{
		"store-test-1": true,
		"store-test-2": false,
	}
	enabledPlugins := map[string]bool{
		"store-test-1": true,
		"store-test-2": false,
	}
	store.UpdateInstalledStatus(installedPlugins, enabledPlugins)

	// Test 11: Load from cache
	err = store.LoadFromCache()
	if err != nil {
		t.Fatalf("Failed to load from cache: %v", err)
	}

	// Verify cache was loaded
	cachedPlugins := store.GetPlugins()
	if len(cachedPlugins) != 3 {
		t.Errorf("Expected 3 cached plugins, got %d", len(cachedPlugins))
	}
}
