package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cod-e-Codes/marchat/plugin/sdk"
)

func TestNewStore(t *testing.T) {
	registryURL := "https://example.com/registry.json"
	cacheDir := "/tmp/test-cache"

	store := NewStore(registryURL, cacheDir)

	if store == nil {
		t.Fatal("NewStore returned nil")
		return
	}

	if store.registryURL != registryURL {
		t.Errorf("Expected registryURL %s, got %s", registryURL, store.registryURL)
	}

	expectedCacheFile := filepath.Join(cacheDir, "store_cache.json")
	if store.cacheFile != expectedCacheFile {
		t.Errorf("Expected cacheFile %s, got %s", expectedCacheFile, store.cacheFile)
	}
}

func TestRefreshWithLocalFile(t *testing.T) {
	// Create a temporary registry file
	registryFile := filepath.Join(t.TempDir(), "registry.json")

	// Create test plugins
	plugins := []StorePlugin{
		{
			Name:        "test-plugin-1",
			Version:     "1.0.0",
			Description: "Test plugin 1",
			Author:      "Test Author",
			License:     "MIT",
			DownloadURL: "https://example.com/plugin1.zip",
			Category:    "utility",
			Tags:        []string{"test", "utility"},
			Commands: []sdk.PluginCommand{
				{
					Name:        "test1",
					Description: "Test command 1",
					Usage:       ":test1",
					AdminOnly:   false,
				},
			},
		},
		{
			Name:        "test-plugin-2",
			Version:     "2.0.0",
			Description: "Test plugin 2",
			Author:      "Test Author 2",
			License:     "Apache-2.0",
			DownloadURL: "https://example.com/plugin2.zip",
			Category:    "fun",
			Tags:        []string{"test", "fun"},
			GoOS:        "linux",
			GoArch:      "amd64",
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

	// Create store with local file URL
	absPath, err := filepath.Abs(registryFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	store := NewStore("file://"+absPath, t.TempDir())

	// Test refresh
	err = store.Refresh()
	if err != nil {
		t.Fatalf("Failed to refresh store: %v", err)
	}

	// Verify plugins were loaded
	loadedPlugins := store.GetPlugins()
	if len(loadedPlugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(loadedPlugins))
	}

	// Verify plugin details
	found := make(map[string]bool)
	for _, plugin := range loadedPlugins {
		found[plugin.Name] = true

		switch plugin.Name {
		case "test-plugin-1":
			if plugin.Version != "1.0.0" {
				t.Errorf("Expected version 1.0.0, got %s", plugin.Version)
			}
			if plugin.Category != "utility" {
				t.Errorf("Expected category utility, got %s", plugin.Category)
			}
			if len(plugin.Tags) != 2 {
				t.Errorf("Expected 2 tags, got %d", len(plugin.Tags))
			}
		case "test-plugin-2":
			if plugin.GoOS != "linux" {
				t.Errorf("Expected GoOS linux, got %s", plugin.GoOS)
			}
			if plugin.GoArch != "amd64" {
				t.Errorf("Expected GoArch amd64, got %s", plugin.GoArch)
			}
		}
	}

	if !found["test-plugin-1"] || !found["test-plugin-2"] {
		t.Error("Not all expected plugins were found")
	}
}

func TestRefreshWithThreeSlashFileURL(t *testing.T) {
	registryFile := filepath.Join(t.TempDir(), "registry.json")
	plugins := []StorePlugin{
		{
			Name:        "slash-test",
			Version:     "1.0.0",
			Description: "Three-slash file URL test",
			Author:      "Test",
			License:     "MIT",
			DownloadURL: "https://example.com/plugin.zip",
			Category:    "test",
		},
	}
	registryData, err := json.MarshalIndent(plugins, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal registry: %v", err)
	}
	if err := os.WriteFile(registryFile, registryData, 0644); err != nil {
		t.Fatalf("Failed to write registry file: %v", err)
	}

	absPath, err := filepath.Abs(registryFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	slashPath := filepath.ToSlash(absPath)
	registryURL := "file://" + slashPath
	if !strings.HasPrefix(slashPath, "/") {
		registryURL = "file:///" + slashPath
	}

	store := NewStore(registryURL, t.TempDir())
	if err := store.Refresh(); err != nil {
		t.Fatalf("Failed to refresh store: %v", err)
	}
	loaded := store.GetPlugins()
	if len(loaded) != 1 || loaded[0].Name != "slash-test" {
		t.Fatalf("expected one plugin slash-test, got %+v", loaded)
	}
}

func TestRefreshWithNewFormat(t *testing.T) {
	// Create a temporary registry file with new format
	registryFile := filepath.Join(t.TempDir(), "registry.json")

	// Create registry with new format
	registry := struct {
		Version string        `json:"version"`
		Plugins []StorePlugin `json:"plugins"`
	}{
		Version: "1.0",
		Plugins: []StorePlugin{
			{
				Name:        "new-format-plugin",
				Version:     "1.0.0",
				Description: "New format plugin",
				Author:      "Test Author",
				License:     "MIT",
				DownloadURL: "https://example.com/plugin.zip",
			},
		},
	}

	// Write registry file
	registryData, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal registry: %v", err)
	}

	if err := os.WriteFile(registryFile, registryData, 0644); err != nil {
		t.Fatalf("Failed to write registry file: %v", err)
	}

	// Create store with local file URL
	absPath, err := filepath.Abs(registryFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	store := NewStore("file://"+absPath, t.TempDir())

	// Test refresh
	err = store.Refresh()
	if err != nil {
		t.Fatalf("Failed to refresh store: %v", err)
	}

	// Verify plugin was loaded
	loadedPlugins := store.GetPlugins()
	if len(loadedPlugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(loadedPlugins))
	}

	if loadedPlugins[0].Name != "new-format-plugin" {
		t.Errorf("Expected plugin name 'new-format-plugin', got %s", loadedPlugins[0].Name)
	}
}

func TestLoadFromCache(t *testing.T) {
	cacheDir := t.TempDir()
	cacheFile := filepath.Join(cacheDir, "store_cache.json")

	// Create test cache data
	plugins := []StorePlugin{
		{
			Name:        "cached-plugin",
			Version:     "1.0.0",
			Description: "Cached plugin",
			Author:      "Test Author",
			License:     "MIT",
			DownloadURL: "https://example.com/plugin.zip",
		},
	}

	cacheData, err := json.MarshalIndent(plugins, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal cache: %v", err)
	}

	if err := os.WriteFile(cacheFile, cacheData, 0644); err != nil {
		t.Fatalf("Failed to write cache file: %v", err)
	}

	// Create store
	store := NewStore("https://example.com/registry.json", cacheDir)

	// Test loading from cache
	err = store.LoadFromCache()
	if err != nil {
		t.Fatalf("Failed to load from cache: %v", err)
	}

	// Verify plugin was loaded
	loadedPlugins := store.GetPlugins()
	if len(loadedPlugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(loadedPlugins))
	}

	if loadedPlugins[0].Name != "cached-plugin" {
		t.Errorf("Expected plugin name 'cached-plugin', got %s", loadedPlugins[0].Name)
	}
}

func TestLoadFromCacheNonExistent(t *testing.T) {
	cacheDir := t.TempDir()

	// Create store with non-existent cache
	store := NewStore("https://example.com/registry.json", cacheDir)

	// Test loading from non-existent cache - should not error
	err := store.LoadFromCache()
	if err != nil {
		t.Fatalf("Expected no error when loading non-existent cache, got %v", err)
	}

	// Should have no plugins
	plugins := store.GetPlugins()
	if len(plugins) != 0 {
		t.Errorf("Expected 0 plugins, got %d", len(plugins))
	}
}

func TestResolvePlugin(t *testing.T) {
	// Create store with test plugins
	plugins := []StorePlugin{
		{
			Name:        "test-plugin",
			Version:     "1.0.0",
			Description: "Test plugin",
			Author:      "Test Author",
			License:     "MIT",
			DownloadURL: "https://example.com/plugin.zip",
			GoOS:        "linux",
			GoArch:      "amd64",
		},
		{
			Name:        "test-plugin",
			Version:     "1.0.0",
			Description: "Test plugin",
			Author:      "Test Author",
			License:     "MIT",
			DownloadURL: "https://example.com/plugin-windows.zip",
			GoOS:        "windows",
			GoArch:      "amd64",
		},
		{
			Name:        "other-plugin",
			Version:     "2.0.0",
			Description: "Other plugin",
			Author:      "Other Author",
			License:     "Apache-2.0",
			DownloadURL: "https://example.com/other.zip",
		},
	}

	store := &Store{plugins: plugins}

	// Test resolving existing plugin with exact match
	plugin := store.ResolvePlugin("test-plugin", "linux", "amd64")
	if plugin == nil {
		t.Fatal("Expected to find plugin with exact match")
		return
	}
	if plugin.GoOS != "linux" || plugin.GoArch != "amd64" {
		t.Errorf("Expected linux/amd64 plugin, got %s/%s", plugin.GoOS, plugin.GoArch)
	}

	// Test resolving existing plugin with OS match only
	plugin = store.ResolvePlugin("test-plugin", "windows", "arm64")
	if plugin == nil {
		t.Fatal("Expected to find plugin with OS match")
		return
	}
	if plugin.GoOS != "windows" {
		t.Errorf("Expected windows plugin, got %s", plugin.GoOS)
	}

	// Test resolving non-existent plugin
	plugin = store.ResolvePlugin("non-existent", "", "")
	if plugin != nil {
		t.Fatal("Expected nil for non-existent plugin")
	}

	// Test resolving plugin without platform info
	plugin = store.ResolvePlugin("other-plugin", "", "")
	if plugin == nil {
		t.Fatal("Expected to find plugin without platform info")
		return
	}
	if plugin.Name != "other-plugin" {
		t.Errorf("Expected 'other-plugin', got %s", plugin.Name)
	}
}

func TestGetPlugin(t *testing.T) {
	plugins := []StorePlugin{
		{
			Name:        "test-plugin",
			Version:     "1.0.0",
			Description: "Test plugin",
			Author:      "Test Author",
			License:     "MIT",
		},
	}

	store := &Store{plugins: plugins}

	// Test getting existing plugin
	plugin := store.GetPlugin("test-plugin")
	if plugin == nil {
		t.Fatal("Expected to find plugin")
		return
	}
	if plugin.Name != "test-plugin" {
		t.Errorf("Expected 'test-plugin', got %s", plugin.Name)
	}

	// Test getting non-existent plugin
	plugin = store.GetPlugin("non-existent")
	if plugin != nil {
		t.Fatal("Expected nil for non-existent plugin")
	}
}

func TestFilterPlugins(t *testing.T) {
	plugins := []StorePlugin{
		{
			Name:        "utility-plugin",
			Version:     "1.0.0",
			Description: "A utility plugin",
			Author:      "Utility Author",
			License:     "MIT",
			Category:    "utility",
			Tags:        []string{"tool", "helper"},
		},
		{
			Name:        "fun-plugin",
			Version:     "2.0.0",
			Description: "A fun plugin",
			Author:      "Fun Author",
			License:     "Apache-2.0",
			Category:    "fun",
			Tags:        []string{"game", "entertainment"},
		},
		{
			Name:        "another-utility",
			Version:     "1.5.0",
			Description: "Another utility plugin",
			Author:      "Another Author",
			License:     "MIT",
			Category:    "utility",
			Tags:        []string{"tool", "productivity"},
		},
	}

	store := &Store{plugins: plugins}

	// Test filtering by category
	filtered := store.FilterPlugins("utility", "", nil)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 utility plugins, got %d", len(filtered))
	}

	// Test filtering by search term
	filtered = store.FilterPlugins("", "fun", nil)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 plugin matching 'fun', got %d", len(filtered))
	}
	if filtered[0].Name != "fun-plugin" {
		t.Errorf("Expected 'fun-plugin', got %s", filtered[0].Name)
	}

	// Test filtering by tags
	filtered = store.FilterPlugins("", "", []string{"tool"})
	if len(filtered) != 2 {
		t.Errorf("Expected 2 plugins with 'tool' tag, got %d", len(filtered))
	}

	// Test filtering by multiple criteria
	filtered = store.FilterPlugins("utility", "another", []string{"tool"})
	if len(filtered) != 1 {
		t.Errorf("Expected 1 plugin matching all criteria, got %d", len(filtered))
	}
	if filtered[0].Name != "another-utility" {
		t.Errorf("Expected 'another-utility', got %s", filtered[0].Name)
	}
}

func TestGetCategories(t *testing.T) {
	plugins := []StorePlugin{
		{Name: "plugin1", Category: "utility"},
		{Name: "plugin2", Category: "fun"},
		{Name: "plugin3", Category: "utility"},
		{Name: "plugin4", Category: "productivity"},
		{Name: "plugin5", Category: ""}, // Empty category
	}

	store := &Store{plugins: plugins}

	categories := store.GetCategories()
	if len(categories) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(categories))
	}

	// Check that all expected categories are present
	expectedCategories := map[string]bool{
		"utility":      true,
		"fun":          true,
		"productivity": true,
	}

	for _, category := range categories {
		if !expectedCategories[category] {
			t.Errorf("Unexpected category: %s", category)
		}
	}
}

func TestGetTags(t *testing.T) {
	plugins := []StorePlugin{
		{Name: "plugin1", Tags: []string{"tool", "helper"}},
		{Name: "plugin2", Tags: []string{"game", "fun"}},
		{Name: "plugin3", Tags: []string{"tool", "productivity"}},
		{Name: "plugin4", Tags: []string{}}, // Empty tags
	}

	store := &Store{plugins: plugins}

	tags := store.GetTags()
	if len(tags) != 5 {
		t.Errorf("Expected 5 unique tags, got %d", len(tags))
	}

	// Check that all expected tags are present
	expectedTags := map[string]bool{
		"tool":         true,
		"helper":       true,
		"game":         true,
		"fun":          true,
		"productivity": true,
	}

	for _, tag := range tags {
		if !expectedTags[tag] {
			t.Errorf("Unexpected tag: %s", tag)
		}
	}
}

func TestUpdateInstalledStatus(t *testing.T) {
	plugins := []StorePlugin{
		{Name: "plugin1"},
		{Name: "plugin2"},
		{Name: "plugin3"},
	}

	store := &Store{plugins: plugins}

	// Update installed status
	installedPlugins := map[string]bool{
		"plugin1": true,
		"plugin2": false,
		"plugin3": true,
	}

	enabledPlugins := map[string]bool{
		"plugin1": true,
		"plugin2": false,
		"plugin3": false,
	}

	store.UpdateInstalledStatus(installedPlugins, enabledPlugins)

	// Verify status was updated
	updatedPlugins := store.GetPlugins()
	for _, plugin := range updatedPlugins {
		switch plugin.Name {
		case "plugin1":
			if !plugin.Installed || !plugin.Enabled {
				t.Errorf("Plugin1 should be installed and enabled")
			}
		case "plugin2":
			if plugin.Installed || plugin.Enabled {
				t.Errorf("Plugin2 should not be installed or enabled")
			}
		case "plugin3":
			if !plugin.Installed || plugin.Enabled {
				t.Errorf("Plugin3 should be installed but not enabled")
			}
		}
	}
}

func TestGetPluginsPreferredForPlatform(t *testing.T) {
	plugins := []StorePlugin{
		{
			Name:   "test-plugin",
			GoOS:   "linux",
			GoArch: "amd64",
		},
		{
			Name:   "test-plugin",
			GoOS:   "windows",
			GoArch: "amd64",
		},
		{
			Name:   "other-plugin",
			GoOS:   "linux",
			GoArch: "amd64",
		},
		{
			Name:   "other-plugin",
			GoOS:   "darwin",
			GoArch: "arm64",
		},
	}

	store := &Store{plugins: plugins}

	// Test with linux/amd64 preference
	preferred := store.GetPluginsPreferredForPlatform("linux", "amd64")
	if len(preferred) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(preferred))
	}

	// Check that we got the preferred variants
	for _, plugin := range preferred {
		if plugin.Name == "test-plugin" && (plugin.GoOS != "linux" || plugin.GoArch != "amd64") {
			t.Errorf("Expected linux/amd64 variant for test-plugin, got %s/%s", plugin.GoOS, plugin.GoArch)
		}
		if plugin.Name == "other-plugin" && (plugin.GoOS != "linux" || plugin.GoArch != "amd64") {
			t.Errorf("Expected linux/amd64 variant for other-plugin, got %s/%s", plugin.GoOS, plugin.GoArch)
		}
	}
}
