package manager

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
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
	"github.com/Cod-e-Codes/marchat/plugin/store"
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
	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	pluginName := "test-plugin"
	zipPath := writeTempPluginZip(t, pluginName)

	registry := []map[string]interface{}{{
		"name":         pluginName,
		"version":      "1.0.0",
		"description":  "Test plugin",
		"author":       "Test Author",
		"license":      "MIT",
		"download_url": "file://" + zipPath,
		"category":     "test",
	}}
	registryFile := writeRegistry(t, registry)

	manager := NewPluginManager(pluginDir, dataDir, "file://"+registryFile)
	store := manager.GetStore()
	if err := store.LoadFromCache(); err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}
	_ = store.Refresh()

	storePlugin := store.ResolvePlugin(pluginName, "", "")
	if storePlugin == nil {
		t.Fatal("Plugin not found in store")
	}
	if storePlugin.Name != pluginName {
		t.Errorf("Expected plugin name %s, got %s", pluginName, storePlugin.Name)
	}
	if storePlugin.DownloadURL != "file://"+zipPath {
		t.Errorf("Expected download URL %s, got %s", "file://"+zipPath, storePlugin.DownloadURL)
	}
}

func TestInstallPluginWithHTTP(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	pluginName := "http-plugin"
	zipData := buildTestPluginZip(t, pluginName)

	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/registry.json":
			registry := []map[string]interface{}{{
				"name":         pluginName,
				"version":      "1.0.0",
				"description":  "HTTP test plugin",
				"author":       "Test Author",
				"license":      "MIT",
				"download_url": serverURL + "/plugin.zip",
				"category":     "test",
			}}
			registryData, _ := json.Marshal(registry)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(registryData)
		case "/plugin.zip":
			_, _ = w.Write(zipData)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	manager := NewPluginManager(pluginDir, dataDir, server.URL+"/registry.json")
	if err := manager.RefreshStore(); err != nil {
		t.Fatalf("Failed to refresh store: %v", err)
	}

	storePlugin := manager.GetStore().ResolvePlugin(pluginName, "", "")
	if storePlugin == nil {
		t.Fatal("Plugin not found in store")
	}
	if storePlugin.Name != pluginName {
		t.Errorf("Expected plugin name %q, got %s", pluginName, storePlugin.Name)
	}
}

func TestInstallPluginWithPlatformLeavesNoDirOnChecksumFailure(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	pluginName := "bad-install"
	zipPath := writeTempPluginZip(t, pluginName)

	registry := []map[string]interface{}{{
		"name":         pluginName,
		"version":      "1.0.0",
		"description":  "Test plugin",
		"author":       "Test Author",
		"license":      "MIT",
		"download_url": "file://" + filepath.ToSlash(zipPath),
		"checksum":     strings.Repeat("0", 64),
		"category":     "test",
	}}
	registryFile := writeRegistry(t, registry)

	manager := NewPluginManager(pluginDir, dataDir, "file://"+registryFile)
	if err := manager.RefreshStore(); err != nil {
		t.Fatalf("Failed to refresh store: %v", err)
	}

	err := manager.InstallPluginWithPlatform(pluginName, "", "")
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected checksum install error, got %v", err)
	}
	assertPluginDirAbsent(t, filepath.Join(pluginDir, pluginName))
}

func TestDownloadPluginRejectsHTTPChecksumMismatchBeforeExtract(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	pluginName := "bad-plugin"
	zipData := buildTestPluginZip(t, pluginName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(zipData)
	}))
	defer server.Close()

	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")
	pluginPath := filepath.Join(pluginDir, pluginName)
	err := manager.downloadPlugin(storePluginForTest(pluginName, server.URL+"/plugin.zip", strings.Repeat("0", 64)), pluginPath)
	if err == nil || !strings.Contains(err.Error(), "checksum validation failed") {
		t.Fatalf("expected checksum validation error, got %v", err)
	}
	assertPluginNotExtracted(t, pluginPath)
}

func TestDownloadPluginRejectsLocalFileChecksumMismatchBeforeExtract(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	pluginName := "bad-local"
	zipData := buildTestPluginZip(t, pluginName)

	zipPath := filepath.Join(t.TempDir(), "plugin.zip")
	if err := os.WriteFile(zipPath, zipData, 0644); err != nil {
		t.Fatalf("failed to write plugin zip: %v", err)
	}

	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")
	pluginPath := filepath.Join(pluginDir, pluginName)
	err := manager.downloadPlugin(storePluginForTest(pluginName, "file://"+filepath.ToSlash(zipPath), strings.Repeat("f", 64)), pluginPath)
	if err == nil || !strings.Contains(err.Error(), "checksum validation failed") {
		t.Fatalf("expected checksum validation error, got %v", err)
	}
	assertPluginNotExtracted(t, pluginPath)
}

func TestDownloadPluginAcceptsLocalFileURL(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	pluginName := "local-plugin"
	zipData := buildTestPluginZip(t, pluginName)

	zipPath := filepath.Join(t.TempDir(), "plugin.zip")
	if err := os.WriteFile(zipPath, zipData, 0644); err != nil {
		t.Fatalf("failed to write plugin zip: %v", err)
	}

	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")
	pluginPath := filepath.Join(pluginDir, pluginName)
	if err := manager.downloadPlugin(storePluginForTest(pluginName, "file://"+filepath.ToSlash(zipPath), ""), pluginPath); err != nil {
		t.Fatalf("downloadPlugin failed for local file URL: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pluginPath, "plugin.json")); err != nil {
		t.Fatalf("expected plugin manifest to be extracted: %v", err)
	}
	assertPluginBinaryExecutable(t, filepath.Join(pluginPath, pluginName))
}

func TestDownloadPluginSetsTarBinaryExecutable(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	pluginName := "tar-plugin"
	tarPath := writeTempPluginTarGz(t, pluginName)

	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")
	pluginPath := filepath.Join(pluginDir, pluginName)
	if err := manager.downloadPlugin(storePluginForTest(pluginName, "file://"+filepath.ToSlash(tarPath), ""), pluginPath); err != nil {
		t.Fatalf("downloadPlugin failed for tar.gz plugin: %v", err)
	}
	assertPluginBinaryExecutable(t, filepath.Join(pluginPath, pluginName))
}

func TestDownloadPluginChmodsOnlyExactBinaryName(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix execute bit check")
	}

	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	pluginName := "my-plugin"
	zipPath := writeTempZipWithEntries(t, map[string][]byte{
		"not-" + pluginName: []byte("decoy"),
		pluginName:          []byte("#!/bin/sh\necho test\n"),
	})

	manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")
	pluginPath := filepath.Join(pluginDir, pluginName)
	if err := manager.downloadPlugin(storePluginForTest(pluginName, "file://"+filepath.ToSlash(zipPath), ""), pluginPath); err != nil {
		t.Fatalf("downloadPlugin failed: %v", err)
	}

	decoyInfo, err := os.Stat(filepath.Join(pluginPath, "not-"+pluginName))
	if err != nil {
		t.Fatalf("expected decoy file: %v", err)
	}
	if decoyInfo.Mode()&0111 != 0 {
		t.Fatal("expected decoy file to remain non-executable")
	}
	assertPluginBinaryExecutable(t, filepath.Join(pluginPath, pluginName))
}

func TestArchiveFilePathRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	unsafe := []string{
		"../escape",
		"nested/../../escape",
		`..\escape`,
		`C:\escape`,
		"C:/escape",
		"nested/C:/escape",
		"//server/share/escape",
		`\\server\share\escape`,
	}
	for _, name := range unsafe {
		if _, err := archiveFilePath(root, name); err == nil {
			t.Fatalf("expected unsafe path error for %q", name)
		}
	}
}

func TestDownloadPluginRejectsArchiveTraversal(t *testing.T) {
	tests := []struct {
		name        string
		downloadURL string
	}{
		{name: "zip parent", downloadURL: writeTempZip(t, "../escape")},
		{name: "zip nested parent", downloadURL: writeTempZip(t, "nested/../../escape")},
		{name: "zip backslash parent", downloadURL: writeTempZip(t, `..\escape`)},
		{name: "zip drive letter", downloadURL: writeTempZip(t, `C:\escape`)},
		{name: "zip drive letter slash", downloadURL: writeTempZip(t, "C:/escape")},
		{name: "zip nested drive letter", downloadURL: writeTempZip(t, "nested/C:/escape")},
		{name: "zip unc", downloadURL: writeTempZip(t, `\\server\share\escape`)},
		{name: "tar parent", downloadURL: writeTempTarGz(t, "../escape")},
		{name: "tar drive letter", downloadURL: writeTempTarGz(t, "C:/escape")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pluginDir := t.TempDir()
			dataDir := t.TempDir()
			pluginName := "unsafe-plugin"
			manager := NewPluginManager(pluginDir, dataDir, "https://example.com/registry.json")
			pluginPath := filepath.Join(pluginDir, pluginName)

			err := manager.downloadPlugin(storePluginForTest(pluginName, tt.downloadURL, ""), pluginPath)
			if err == nil || !strings.Contains(err.Error(), "unsafe file path in archive") {
				t.Fatalf("expected unsafe path error, got %v", err)
			}
			assertPluginNotExtracted(t, pluginPath)
			if _, err := os.Stat(filepath.Join(pluginDir, "escape")); !os.IsNotExist(err) {
				t.Fatalf("expected escape file not to exist, stat err=%v", err)
			}
		})
	}
}

func TestReplacePluginDirRestoresExistingPluginOnFailure(t *testing.T) {
	root := t.TempDir()
	pluginPath := filepath.Join(root, "plugin")
	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}
	marker := filepath.Join(pluginPath, "plugin.json")
	if err := os.WriteFile(marker, []byte(`{"name":"plugin"}`), 0644); err != nil {
		t.Fatalf("failed to write marker: %v", err)
	}

	err := replacePluginDir(pluginPath, filepath.Join(root, "missing-staging"))
	if err == nil || !strings.Contains(err.Error(), "failed to install staged plugin") {
		t.Fatalf("expected staged install error, got %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("expected existing plugin to be restored: %v", err)
	}
}

func TestOpenPluginDownloadUsesLocalFileDirectly(t *testing.T) {
	zipPath := writeTempPluginZip(t, "local-plugin")
	download, err := openPluginDownload("file://" + filepath.ToSlash(zipPath))
	if err != nil {
		t.Fatalf("openPluginDownload failed: %v", err)
	}
	defer download.Close()
	if download.remove {
		t.Fatal("local plugin downloads should not be copied to removable temp files")
	}
	if download.file.Name() != zipPath {
		t.Fatalf("expected local file %q, got %q", zipPath, download.file.Name())
	}
}

func storePluginForTest(name, downloadURL, checksum string) *store.StorePlugin {
	return &store.StorePlugin{Name: name, DownloadURL: downloadURL, Checksum: checksum}
}

func buildTestPluginZip(t *testing.T, pluginName string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	binaryWriter, err := zipWriter.Create(pluginName)
	if err != nil {
		t.Fatalf("failed to create binary ZIP entry: %v", err)
	}
	if _, err := binaryWriter.Write([]byte("#!/bin/sh\necho test\n")); err != nil {
		t.Fatalf("failed to write binary ZIP entry: %v", err)
	}

	manifest := sdk.PluginManifest{Name: pluginName, Version: "1.0.0", Description: "Test plugin", Author: "Test Author", License: "MIT"}
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}
	manifestWriter, err := zipWriter.Create("plugin.json")
	if err != nil {
		t.Fatalf("failed to create manifest ZIP entry: %v", err)
	}
	if _, err := manifestWriter.Write(manifestData); err != nil {
		t.Fatalf("failed to write manifest ZIP entry: %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("failed to close ZIP writer: %v", err)
	}
	return buf.Bytes()
}

func writeTempPluginZip(t *testing.T, pluginName string) string {
	t.Helper()
	zipPath := filepath.Join(t.TempDir(), pluginName+".zip")
	if err := os.WriteFile(zipPath, buildTestPluginZip(t, pluginName), 0644); err != nil {
		t.Fatalf("failed to write plugin zip: %v", err)
	}
	return zipPath
}

func writeRegistry(t *testing.T, registry []map[string]interface{}) string {
	t.Helper()
	registryFile := filepath.Join(t.TempDir(), "registry.json")
	registryData, err := json.Marshal(registry)
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
	return filepath.ToSlash(absPath)
}

func writeTempZip(t *testing.T, name string) string {
	t.Helper()
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	writer, err := zipWriter.Create(name)
	if err != nil {
		t.Fatalf("failed to create ZIP entry: %v", err)
	}
	if _, err := writer.Write([]byte("escape")); err != nil {
		t.Fatalf("failed to write ZIP entry: %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("failed to close ZIP writer: %v", err)
	}
	zipPath := filepath.Join(t.TempDir(), "unsafe.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write unsafe zip: %v", err)
	}
	return "file://" + filepath.ToSlash(zipPath)
}

func writeTempTarGz(t *testing.T, name string) string {
	t.Helper()
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)
	data := []byte("escape")
	if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(data))}); err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}
	if _, err := tarWriter.Write(data); err != nil {
		t.Fatalf("failed to write tar entry: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	tarPath := filepath.Join(t.TempDir(), "unsafe.tar.gz")
	if err := os.WriteFile(tarPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write unsafe tar: %v", err)
	}
	return "file://" + filepath.ToSlash(tarPath)
}

func writeTempPluginTarGz(t *testing.T, pluginName string) string {
	t.Helper()
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	binaryData := []byte("#!/bin/sh\necho test\n")
	if err := tarWriter.WriteHeader(&tar.Header{Name: pluginName, Mode: 0644, Size: int64(len(binaryData))}); err != nil {
		t.Fatalf("failed to write tar binary header: %v", err)
	}
	if _, err := tarWriter.Write(binaryData); err != nil {
		t.Fatalf("failed to write tar binary: %v", err)
	}

	manifest := sdk.PluginManifest{Name: pluginName, Version: "1.0.0", Description: "Test plugin", Author: "Test Author", License: "MIT"}
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}
	if err := tarWriter.WriteHeader(&tar.Header{Name: "plugin.json", Mode: 0644, Size: int64(len(manifestData))}); err != nil {
		t.Fatalf("failed to write tar manifest header: %v", err)
	}
	if _, err := tarWriter.Write(manifestData); err != nil {
		t.Fatalf("failed to write tar manifest: %v", err)
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	tarPath := filepath.Join(t.TempDir(), pluginName+".tar.gz")
	if err := os.WriteFile(tarPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write plugin tar: %v", err)
	}
	return tarPath
}

func writeTempZipWithEntries(t *testing.T, entries map[string][]byte) string {
	t.Helper()
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	for name, data := range entries {
		writer, err := zipWriter.Create(name)
		if err != nil {
			t.Fatalf("failed to create ZIP entry %q: %v", name, err)
		}
		if _, err := writer.Write(data); err != nil {
			t.Fatalf("failed to write ZIP entry %q: %v", name, err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("failed to close ZIP writer: %v", err)
	}
	zipPath := filepath.Join(t.TempDir(), "entries.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write zip: %v", err)
	}
	return zipPath
}

func assertPluginBinaryExecutable(t *testing.T, binaryPath string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(binaryPath)
	if err != nil {
		t.Fatalf("expected plugin binary at %s: %v", binaryPath, err)
	}
	if info.Mode()&0111 == 0 {
		t.Fatalf("expected plugin binary %s to be executable, mode=%v", binaryPath, info.Mode())
	}
}

func assertPluginDirAbsent(t *testing.T, pluginPath string) {
	t.Helper()
	if _, err := os.Stat(pluginPath); !os.IsNotExist(err) {
		t.Fatalf("expected plugin directory absent, stat err=%v", err)
	}
}

func assertPluginNotExtracted(t *testing.T, pluginPath string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(pluginPath, "plugin.json")); !os.IsNotExist(err) {
		t.Fatalf("expected plugin manifest not to be extracted, stat err=%v", err)
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
