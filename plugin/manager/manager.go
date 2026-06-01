package manager

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Cod-e-Codes/marchat/plugin/host"
	"github.com/Cod-e-Codes/marchat/plugin/license"
	"github.com/Cod-e-Codes/marchat/plugin/sdk"
	"github.com/Cod-e-Codes/marchat/plugin/store"
	"github.com/Cod-e-Codes/marchat/shared"
)

const (
	pluginDownloadTimeout = 60 * time.Second
	maxPluginDownloadSize = 100 * 1024 * 1024 // 100 MB
)

var pluginHTTPClient = &http.Client{Timeout: pluginDownloadTimeout}

// Valid plugin name pattern: lowercase letters, numbers, hyphens, underscores only
var validPluginNameRegex = regexp.MustCompile(`^[a-z0-9_-]+$`)

// validatePluginName ensures plugin names are safe and cannot cause path traversal
func validatePluginName(name string) error {
	if name == "" {
		return errors.New("plugin name cannot be empty")
	}
	if len(name) > 64 {
		return errors.New("plugin name too long (max 64 characters)")
	}
	if !validPluginNameRegex.MatchString(name) {
		return errors.New("plugin name must contain only lowercase letters, numbers, hyphens, and underscores")
	}
	if strings.Contains(name, "..") {
		return errors.New("plugin name cannot contain '..'")
	}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") {
		return errors.New("plugin name cannot start with path separator")
	}
	return nil
}

// PluginState represents the persisted state of plugins
type PluginState struct {
	Enabled map[string]bool `json:"enabled"` // plugin name -> enabled status
}

// PluginManager manages plugin installation and commands
type PluginManager struct {
	host             *host.PluginHost
	store            *store.Store
	pluginDir        string
	dataDir          string
	registryURL      string
	stateFile        string
	licenseValidator *license.LicenseValidator
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(pluginDir, dataDir, registryURL string) *PluginManager {
	host := host.NewPluginHost(pluginDir, dataDir)
	store := store.NewStore(registryURL, dataDir)

	pm := &PluginManager{
		host:        host,
		store:       store,
		pluginDir:   pluginDir,
		dataDir:     dataDir,
		registryURL: registryURL,
		stateFile:   filepath.Join(dataDir, "plugin_state.json"),
	}

	// Auto-discover and load installed plugins
	pm.discoverInstalledPlugins()

	return pm
}

// loadPluginState loads the persisted plugin state
func (pm *PluginManager) loadPluginState() *PluginState {
	data, err := os.ReadFile(pm.stateFile)
	if err != nil {
		// State file doesn't exist - return default state
		return &PluginState{
			Enabled: make(map[string]bool),
		}
	}

	var state PluginState
	if err := json.Unmarshal(data, &state); err != nil {
		// Corrupted state file - return default
		return &PluginState{
			Enabled: make(map[string]bool),
		}
	}

	if state.Enabled == nil {
		state.Enabled = make(map[string]bool)
	}

	return &state
}

// savePluginState persists the current plugin state
func (pm *PluginManager) savePluginState() error {
	state := &PluginState{
		Enabled: make(map[string]bool),
	}

	// Collect enabled status from all plugins
	for name, instance := range pm.host.ListPlugins() {
		state.Enabled[name] = instance.Enabled
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(pm.stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// discoverInstalledPlugins scans the plugin directory and loads all installed plugins
func (pm *PluginManager) discoverInstalledPlugins() {
	// Load persisted plugin state
	state := pm.loadPluginState()

	// Read plugin directory
	entries, err := os.ReadDir(pm.pluginDir)
	if err != nil {
		// Plugin directory doesn't exist or can't be read - not an error on first run
		return
	}

	// Load each plugin directory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginName := entry.Name()

		// Validate plugin name
		if err := validatePluginName(pluginName); err != nil {
			continue
		}

		// Check if plugin.json exists
		manifestPath := filepath.Join(pm.pluginDir, pluginName, "plugin.json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			continue
		}

		// Load plugin
		if err := pm.host.LoadPlugin(pluginName); err != nil {
			continue
		}

		if instance := pm.host.GetPlugin(pluginName); instance != nil && instance.Manifest != nil {
			if err := pm.validateManifestConstraints(pluginName, instance.Manifest); err != nil {
				log.Printf("Plugin %s skipped: %v", pluginName, err)
				pm.host.UnloadPlugin(pluginName)
				continue
			}
			pm.checkLicense(pluginName, instance.Manifest)
		}

		// Set enabled status from saved state (default to true if not found)
		enabled, exists := state.Enabled[pluginName]
		if !exists {
			enabled = true // New plugins default to enabled
		}

		instance := pm.host.GetPlugin(pluginName)
		if instance != nil {
			instance.Enabled = enabled

			// Auto-start enabled plugins
			if enabled {
				_ = pm.host.StartPlugin(pluginName)
			}
		}
	}
}

// InstallPlugin installs a plugin from the store using the current platform
func (pm *PluginManager) InstallPlugin(name string) error {
	// Validate plugin name to prevent path traversal
	if err := validatePluginName(name); err != nil {
		return fmt.Errorf("invalid plugin name: %w", err)
	}
	return pm.InstallPluginWithPlatform(name, "", "")
}

// InstallPluginWithPlatform installs a plugin selecting a specific os/arch if provided.
// When osName or arch are empty, the current runtime platform is used for selection.
func (pm *PluginManager) InstallPluginWithPlatform(name, osName, arch string) error {
	// Validate plugin name to prevent path traversal
	if err := validatePluginName(name); err != nil {
		return fmt.Errorf("invalid plugin name: %w", err)
	}

	// Get plugin from store
	plugin := pm.store.ResolvePlugin(name, osName, arch)
	if plugin == nil {
		return fmt.Errorf("plugin %s not found in store", name)
	}

	// Create plugin directory
	pluginPath := filepath.Join(pm.pluginDir, name)
	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Download plugin
	if err := pm.downloadPlugin(plugin, pluginPath); err != nil {
		return fmt.Errorf("failed to download plugin: %w", err)
	}

	// Load plugin into host
	if err := pm.host.LoadPlugin(name); err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	if instance := pm.host.GetPlugin(name); instance != nil && instance.Manifest != nil {
		if err := pm.validateManifestConstraints(name, instance.Manifest); err != nil {
			pm.host.UnloadPlugin(name)
			return err
		}
		pm.checkLicense(name, instance.Manifest)
	}

	// Start plugin
	if err := pm.host.StartPlugin(name); err != nil {
		return fmt.Errorf("failed to start plugin: %w", err)
	}

	return nil
}

// UninstallPlugin removes a plugin
func (pm *PluginManager) UninstallPlugin(name string) error {
	// Validate plugin name to prevent path traversal
	if err := validatePluginName(name); err != nil {
		return fmt.Errorf("invalid plugin name: %w", err)
	}

	// Stop plugin if running
	if err := pm.host.StopPlugin(name); err != nil {
		return fmt.Errorf("failed to stop plugin: %w", err)
	}

	// Unload plugin from host to release all references
	pm.host.UnloadPlugin(name)

	// Remove plugin directory
	pluginPath := filepath.Join(pm.pluginDir, name)
	if err := os.RemoveAll(pluginPath); err != nil {
		return fmt.Errorf("failed to remove plugin directory: %w", err)
	}

	// Remove data directory
	dataPath := filepath.Join(pm.dataDir, name)
	if err := os.RemoveAll(dataPath); err != nil {
		return fmt.Errorf("failed to remove plugin data: %w", err)
	}

	return nil
}

// EnablePlugin enables a plugin
func (pm *PluginManager) EnablePlugin(name string) error {
	// Validate plugin name to prevent path traversal
	if err := validatePluginName(name); err != nil {
		return fmt.Errorf("invalid plugin name: %w", err)
	}
	if err := pm.host.EnablePlugin(name); err != nil {
		return err
	}
	// Save state after enabling
	_ = pm.savePluginState()
	return nil
}

// DisablePlugin disables a plugin
func (pm *PluginManager) DisablePlugin(name string) error {
	// Validate plugin name to prevent path traversal
	if err := validatePluginName(name); err != nil {
		return fmt.Errorf("invalid plugin name: %w", err)
	}
	if err := pm.host.DisablePlugin(name); err != nil {
		return err
	}
	// Save state after disabling
	_ = pm.savePluginState()
	return nil
}

// ListPlugins returns all installed plugins
func (pm *PluginManager) ListPlugins() map[string]*host.PluginInstance {
	return pm.host.ListPlugins()
}

// GetPlugin returns a specific plugin
func (pm *PluginManager) GetPlugin(name string) *host.PluginInstance {
	// Validate plugin name to prevent path traversal
	if err := validatePluginName(name); err != nil {
		return nil
	}
	return pm.host.GetPlugin(name)
}

// ExecuteCommand executes a plugin command
func (pm *PluginManager) ExecuteCommand(pluginName, command string, args []string) error {
	// Validate plugin name to prevent path traversal
	if err := validatePluginName(pluginName); err != nil {
		return fmt.Errorf("invalid plugin name: %w", err)
	}
	return pm.host.ExecuteCommand(pluginName, command, args)
}

// SendMessage sends a message to all enabled plugins
func (pm *PluginManager) SendMessage(msg sdk.Message) {
	pm.host.SendMessage(msg)
}

// GetMessageChannel returns the channel for receiving messages from plugins
func (pm *PluginManager) GetMessageChannel() <-chan sdk.Message {
	return pm.host.GetMessageChannel()
}

// UpdateUserList updates the user list for plugins
func (pm *PluginManager) UpdateUserList(users []string) {
	pm.host.UpdateUserList(users)
}

// RefreshStore refreshes the plugin store
func (pm *PluginManager) RefreshStore() error {
	return pm.store.Refresh()
}

// LoadStoreFromCache loads the store from cache
func (pm *PluginManager) LoadStoreFromCache() error {
	return pm.store.LoadFromCache()
}

// GetStore returns the plugin store
func (pm *PluginManager) GetStore() *store.Store {
	return pm.store
}

// downloadPlugin downloads a plugin from the given URL
func (pm *PluginManager) downloadPlugin(plugin *store.StorePlugin, pluginPath string) error {
	download, err := openPluginDownload(plugin.DownloadURL)
	if err != nil {
		return err
	}
	defer download.Close()

	if plugin.Checksum != "" {
		if err := pm.validateDownloadChecksum(download.file, plugin.Checksum); err != nil {
			return fmt.Errorf("checksum validation failed: %w", err)
		}
	}
	if _, err := download.file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to rewind plugin download: %w", err)
	}

	archivePath := plugin.DownloadURL
	if parsedURL, err := url.Parse(plugin.DownloadURL); err == nil && parsedURL.Path != "" {
		archivePath = parsedURL.Path
	}
	return pm.extractPluginDownload(download.file, archivePath, pluginPath, plugin.Name)
}

type pluginDownload struct {
	file   *os.File
	remove bool
}

func (d *pluginDownload) Close() error {
	if d == nil || d.file == nil {
		return nil
	}
	name := d.file.Name()
	err := d.file.Close()
	if d.remove {
		if removeErr := os.Remove(name); err == nil && removeErr != nil {
			err = removeErr
		}
	}
	return err
}

func openPluginDownload(downloadURL string) (*pluginDownload, error) {
	if parsedURL, err := url.Parse(downloadURL); err == nil && parsedURL.Scheme == "file" {
		return openLocalPluginDownload(parsedURL)
	}
	return openRemotePluginDownload(downloadURL)
}

func openLocalPluginDownload(parsedURL *url.URL) (*pluginDownload, error) {
	filePath, err := fileURLPath(parsedURL)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open local plugin file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat local plugin file: %w", err)
	}
	if info.IsDir() {
		file.Close()
		return nil, fmt.Errorf("local plugin file is a directory")
	}
	if info.Size() > maxPluginDownloadSize {
		file.Close()
		return nil, fmt.Errorf("plugin download exceeds maximum size of %d bytes", maxPluginDownloadSize)
	}
	return &pluginDownload{file: file}, nil
}

func openRemotePluginDownload(downloadURL string) (*pluginDownload, error) {
	resp, err := pluginHTTPClient.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download plugin: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "plugin-download-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	ok := false
	defer func() {
		if !ok {
			name := tmp.Name()
			tmp.Close()
			_ = os.Remove(name)
		}
	}()

	written, err := io.Copy(tmp, io.LimitReader(resp.Body, maxPluginDownloadSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin download: %w", err)
	}
	if written > maxPluginDownloadSize {
		return nil, fmt.Errorf("plugin download exceeds maximum size of %d bytes", maxPluginDownloadSize)
	}
	if _, err := tmp.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to rewind plugin download: %w", err)
	}

	ok = true
	return &pluginDownload{file: tmp, remove: true}, nil
}

func fileURLPath(parsedURL *url.URL) (string, error) {
	if parsedURL == nil || parsedURL.Scheme != "file" {
		return "", fmt.Errorf("not a file URL")
	}

	host := parsedURL.Host
	if host == "localhost" {
		host = ""
	}

	var filePath string
	switch {
	case host == "":
		var err error
		filePath, err = url.PathUnescape(parsedURL.Path)
		if err != nil {
			return "", fmt.Errorf("invalid file URL path: %w", err)
		}
	case len(host) == 2 && host[1] == ':':
		// file://C:/path on Windows (host is the drive letter)
		filePath = host + parsedURL.Path
	default:
		return "", fmt.Errorf("unsupported file URL host %q", parsedURL.Host)
	}

	if filePath == "" {
		return "", fmt.Errorf("file URL path cannot be empty")
	}
	if len(filePath) >= 3 && filePath[0] == '/' && filePath[2] == ':' {
		filePath = filePath[1:]
	}
	return filepath.FromSlash(filePath), nil
}

func (pm *PluginManager) extractPluginDownload(download *os.File, archivePath, pluginPath, pluginName string) error {
	parentDir := filepath.Dir(pluginPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin parent directory: %w", err)
	}

	stagingDir, err := os.MkdirTemp(parentDir, ".plugin-staging-*")
	if err != nil {
		return fmt.Errorf("failed to create plugin staging directory: %w", err)
	}
	staged := false
	defer func() {
		if !staged {
			_ = os.RemoveAll(stagingDir)
		}
	}()

	switch {
	case strings.HasSuffix(archivePath, ".zip"):
		err = pm.extractZip(download, stagingDir, pluginName)
	case strings.HasSuffix(archivePath, ".tar.gz"), strings.HasSuffix(archivePath, ".tgz"):
		err = pm.extractTarGz(download, stagingDir, pluginName)
	default:
		err = pm.downloadBinary(download, stagingDir, pluginName)
	}
	if err != nil {
		return err
	}
	if err := replacePluginDir(pluginPath, stagingDir); err != nil {
		return err
	}
	staged = true
	return nil
}

func replacePluginDir(pluginPath, stagingDir string) error {
	if _, err := os.Stat(pluginPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to inspect existing plugin directory: %w", err)
		}
		if err := os.Rename(stagingDir, pluginPath); err != nil {
			return fmt.Errorf("failed to install staged plugin: %w", err)
		}
		return nil
	}

	backupDir, err := os.MkdirTemp(filepath.Dir(pluginPath), ".plugin-backup-*")
	if err != nil {
		return fmt.Errorf("failed to create plugin backup path: %w", err)
	}
	if err := os.Remove(backupDir); err != nil {
		return fmt.Errorf("failed to prepare plugin backup path: %w", err)
	}
	backupActive := false
	defer func() {
		if backupActive {
			if err := os.Rename(backupDir, pluginPath); err != nil {
				return
			}
		}
		_ = os.RemoveAll(backupDir)
	}()

	if err := os.Rename(pluginPath, backupDir); err != nil {
		return fmt.Errorf("failed to back up existing plugin directory: %w", err)
	}
	backupActive = true
	if err := os.Rename(stagingDir, pluginPath); err != nil {
		return fmt.Errorf("failed to install staged plugin: %w", err)
	}
	backupActive = false
	return nil
}

func archiveFilePath(root, name string) (string, error) {
	original := name
	name = strings.ReplaceAll(name, `\`, "/")
	if name == "" || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("unsafe file path in archive: %s", original)
	}

	for _, part := range strings.Split(name, "/") {
		if part == ".." || strings.Contains(part, ":") || filepath.VolumeName(part) != "" {
			return "", fmt.Errorf("unsafe file path in archive: %s", original)
		}
	}

	clean := path.Clean(name)
	if clean == "." {
		return "", fmt.Errorf("unsafe file path in archive: %s", original)
	}

	target := filepath.Join(root, filepath.FromSlash(clean))
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe file path in archive: %s", original)
	}
	return target, nil
}

func archiveEntryBaseName(name string) string {
	return path.Base(strings.ReplaceAll(name, `\`, "/"))
}

// extractZip extracts a zip file.
func (pm *PluginManager) extractZip(file *os.File, destDir, pluginName string) error {
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat zip file: %w", err)
	}
	zipReader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}

	for _, file := range zipReader.File {
		filePath, err := archiveFilePath(destDir, file.Name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}
		if err := extractZipFile(file, filePath); err != nil {
			return err
		}
		if archiveEntryBaseName(file.Name) == pluginName {
			if err := os.Chmod(filePath, 0755); err != nil {
				return fmt.Errorf("failed to make executable: %w", err)
			}
		}
	}
	return nil
}

func extractZipFile(file *zip.File, dst string) error {
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file in zip: %w", err)
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	return nil
}

// extractTarGz extracts a tar.gz file.
func (pm *PluginManager) extractTarGz(reader io.Reader, destDir, pluginName string) error {
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		filePath, err := archiveFilePath(destDir, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filePath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
			if err := extractTarFile(tarReader, filePath); err != nil {
				return err
			}
			if archiveEntryBaseName(header.Name) == pluginName {
				if err := os.Chmod(filePath, 0755); err != nil {
					return fmt.Errorf("failed to make executable: %w", err)
				}
			}
		}
	}
	return nil
}

func extractTarFile(reader io.Reader, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, reader); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	return nil
}

// downloadBinary downloads a single binary file.
func (pm *PluginManager) downloadBinary(reader io.Reader, pluginPath, pluginName string) error {
	binaryPath := filepath.Join(pluginPath, pluginName)
	file, err := os.Create(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to create binary file: %w", err)
	}
	defer file.Close()
	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return fmt.Errorf("failed to make executable: %w", err)
	}
	return nil
}

// validateDownloadChecksum validates the checksum of the downloaded file
func (pm *PluginManager) validateDownloadChecksum(file *os.File, expectedChecksum string) error {
	// Reset file position
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	calculatedChecksum := hex.EncodeToString(hash.Sum(nil))

	// Handle both formats: just hash or "sha256:hash"
	expectedHash := expectedChecksum
	if strings.HasPrefix(expectedChecksum, "sha256:") {
		expectedHash = strings.TrimPrefix(expectedChecksum, "sha256:")
	}

	if calculatedChecksum != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, calculatedChecksum)
	}

	return nil
}

// knownPermissions lists the permission values that plugins may request.
var knownPermissions = map[string]bool{
	"messages": true,
	"commands": true,
	"users":    true,
}

// validateManifestConstraints checks min_version, max_version, and permissions on a manifest.
// Returns an error if the server version is outside the allowed range.
func (pm *PluginManager) validateManifestConstraints(name string, manifest *sdk.PluginManifest) error {
	if manifest == nil {
		return nil
	}

	serverVer := shared.ServerVersion

	if manifest.MinVersion != "" && serverVer != "dev" {
		if shared.CompareVersions(serverVer, manifest.MinVersion) < 0 {
			return fmt.Errorf("plugin %s requires server version >= %s (current: %s)", name, manifest.MinVersion, serverVer)
		}
	}

	if manifest.MaxVersion != "" && serverVer != "dev" {
		if shared.CompareVersions(serverVer, manifest.MaxVersion) > 0 {
			return fmt.Errorf("plugin %s requires server version <= %s (current: %s)", name, manifest.MaxVersion, serverVer)
		}
	}

	for _, perm := range manifest.Permissions {
		if !knownPermissions[perm] {
			log.Printf("Warning: plugin %s requests unknown permission %q", name, perm)
		}
	}

	return nil
}

// SetLicenseValidator sets an optional license validator for commercial plugin enforcement
func (pm *PluginManager) SetLicenseValidator(lv *license.LicenseValidator) {
	pm.licenseValidator = lv
}

// checkLicense logs a warning if a commercial plugin has no valid license.
// It never blocks loading (graceful degradation).
func (pm *PluginManager) checkLicense(name string, manifest *sdk.PluginManifest) {
	if manifest == nil {
		return
	}
	licenseType := strings.ToLower(manifest.License)
	if licenseType != "commercial" && licenseType != "proprietary" {
		return
	}
	if pm.licenseValidator == nil {
		log.Printf("Warning: plugin %s has a %s license but no license validator is configured", name, manifest.License)
		return
	}
	valid, err := pm.licenseValidator.IsLicenseValid(name)
	if err != nil {
		log.Printf("Warning: license check error for plugin %s: %v", name, err)
		return
	}
	if !valid {
		log.Printf("Warning: plugin %s requires a %s license but no valid license was found", name, manifest.License)
	}
}

// GetPluginCommands returns all available plugin commands
func (pm *PluginManager) GetPluginCommands() map[string][]sdk.PluginCommand {
	commands := make(map[string][]sdk.PluginCommand)

	for name, instance := range pm.host.ListPlugins() {
		if instance.Manifest != nil {
			commands[name] = instance.Manifest.Commands
		}
	}

	return commands
}

// GetPluginManifest returns the manifest for a plugin
func (pm *PluginManager) GetPluginManifest(name string) *sdk.PluginManifest {
	// Validate plugin name to prevent path traversal
	if err := validatePluginName(name); err != nil {
		return nil
	}
	instance := pm.host.GetPlugin(name)
	if instance == nil {
		return nil
	}
	return instance.Manifest
}
