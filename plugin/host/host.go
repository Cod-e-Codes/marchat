package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Cod-e-Codes/marchat/plugin/sdk"
)

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

// PluginHost manages the lifecycle and communication with plugins
type PluginHost struct {
	plugins     map[string]*PluginInstance
	pluginDir   string
	dataDir     string
	mu          sync.RWMutex
	messageChan chan sdk.Message
	userList    []string
}

// PluginInstance represents a running plugin
type PluginInstance struct {
	Name     string
	Manifest *sdk.PluginManifest
	Process  *exec.Cmd
	Stdin    io.WriteCloser
	Stdout   io.ReadCloser
	Stderr   io.ReadCloser
	Config   sdk.Config
	Enabled  bool
	decoder  *json.Decoder
	mu       sync.Mutex
}

// NewPluginHost creates a new plugin host
func NewPluginHost(pluginDir, dataDir string) *PluginHost {
	return &PluginHost{
		plugins:     make(map[string]*PluginInstance),
		pluginDir:   pluginDir,
		dataDir:     dataDir,
		messageChan: make(chan sdk.Message, 100),
	}
}

// LoadPlugin loads a plugin from the plugin directory
func (h *PluginHost) LoadPlugin(name string) error {
	// Validate plugin name to prevent path traversal and command injection
	if err := validatePluginName(name); err != nil {
		return fmt.Errorf("invalid plugin name: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	pluginPath := filepath.Join(h.pluginDir, name)
	manifestPath := filepath.Join(pluginPath, "plugin.json")

	// Read and validate manifest
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read plugin manifest: %w", err)
	}

	var manifest sdk.PluginManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to parse plugin manifest: %w", err)
	}

	if err := sdk.ValidateManifest(&manifest); err != nil {
		return fmt.Errorf("invalid plugin manifest: %w", err)
	}

	// Resolve plugin binary path respecting host OS
	var binaryPath string
	if runtime.GOOS == "windows" {
		// Prefer .exe on Windows when no extension is provided
		if filepath.Ext(name) == "" {
			candidateExe := filepath.Join(pluginPath, name+".exe")
			if _, err := os.Stat(candidateExe); err == nil {
				binaryPath = candidateExe
			} else {
				binaryPath = filepath.Join(pluginPath, name)
			}
		} else {
			binaryPath = filepath.Join(pluginPath, name)
		}
	} else {
		// Non-Windows: use name as-is; do not append .exe
		binaryPath = filepath.Join(pluginPath, name)
	}
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin binary not found: %s", binaryPath)
	}

	// Create plugin instance
	instance := &PluginInstance{
		Name:     name,
		Manifest: &manifest,
		Config: sdk.Config{
			PluginDir: pluginPath,
			DataDir:   filepath.Join(h.dataDir, name),
			Settings:  make(map[string]string),
		},
		Enabled: true,
	}

	h.plugins[name] = instance
	return nil
}

// StartPlugin starts a plugin subprocess
func (h *PluginHost) StartPlugin(name string) error {
	// Validate plugin name to prevent path traversal and command injection
	if err := validatePluginName(name); err != nil {
		return fmt.Errorf("invalid plugin name: %w", err)
	}

	h.mu.RLock()
	instance, exists := h.plugins[name]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	instance.mu.Lock()
	defer instance.mu.Unlock()

	if instance.Process != nil {
		return fmt.Errorf("plugin %s is already running", name)
	}

	// Create plugin data directory
	if err := os.MkdirAll(instance.Config.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin data directory: %w", err)
	}

	// Start plugin subprocess - resolve binary path per OS
	var binaryPath string
	if runtime.GOOS == "windows" {
		if filepath.Ext(name) == "" {
			candidateExe := filepath.Join(instance.Config.PluginDir, name+".exe")
			if _, err := os.Stat(candidateExe); err == nil {
				binaryPath = candidateExe
			} else {
				binaryPath = filepath.Join(instance.Config.PluginDir, name)
			}
		} else {
			binaryPath = filepath.Join(instance.Config.PluginDir, name)
		}
	} else {
		binaryPath = filepath.Join(instance.Config.PluginDir, name)
	}

	// Plugin starting with binary path

	// Use absolute path for the executable
	absBinaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	cmd := exec.CommandContext(context.Background(), absBinaryPath)
	cmd.Dir = instance.Config.PluginDir

	// Set up pipes for communication
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	instance.Process = cmd
	instance.Stdin = stdin
	instance.Stdout = stdout
	instance.Stderr = stderr

	// Start the process
	if err := cmd.Start(); err != nil {
		// Provide clearer error on platform mismatch
		errMsg := strings.ToLower(err.Error())
		expected := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
		if strings.Contains(errMsg, "exec format error") ||
			strings.Contains(errMsg, "not a valid win32 application") ||
			strings.Contains(errMsg, "wrong architecture") {
			return fmt.Errorf("plugin %s binary is not compatible with this host (%s). Install the matching build for your platform or use :install %s --os %s --arch %s. underlying error: %v", name, expected, name, runtime.GOOS, runtime.GOARCH, err)
		}
		return fmt.Errorf("failed to start plugin %s: %w", name, err)
	}

	// Initialize plugin
	if err := h.initializePlugin(instance); err != nil {
		// Clean up without calling StopPlugin to avoid deadlock (we already hold instance.mu)
		if instance.Process != nil && instance.Process.Process != nil {
			_ = instance.Process.Process.Kill()
		}
		instance.Process = nil
		instance.Stdin = nil
		instance.Stdout = nil
		instance.Stderr = nil
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	// Start communication goroutines
	go h.handlePluginOutput(instance)
	go h.handlePluginErrors(instance)

	log.Printf("Plugin %s started successfully", name)
	return nil
}

// StopPlugin stops a plugin subprocess
func (h *PluginHost) StopPlugin(name string) error {
	// Validate plugin name to prevent path traversal
	if err := validatePluginName(name); err != nil {
		return fmt.Errorf("invalid plugin name: %w", err)
	}

	h.mu.RLock()
	instance, exists := h.plugins[name]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	instance.mu.Lock()
	defer instance.mu.Unlock()

	if instance.Process == nil {
		return nil // Already stopped
	}

	// Send shutdown request
	shutdownReq := sdk.PluginRequest{
		Type: "shutdown",
	}
	if err := h.sendRequest(instance, shutdownReq); err != nil {
		log.Printf("Failed to send shutdown request to plugin %s: %v", name, err)
	}

	// Wait for graceful shutdown with a short timeout (plugins should shutdown quickly)
	done := make(chan error, 1)
	go func() {
		done <- instance.Process.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("Plugin %s exited with error: %v", name, err)
		}
	case <-time.After(1 * time.Second):
		// Force kill if graceful shutdown takes too long
		// Plugins over JSON IPC should be able to shutdown in under 1 second
		if err := instance.Process.Process.Kill(); err != nil {
			log.Printf("Failed to force kill plugin %s: %v", name, err)
		}
		log.Printf("Plugin %s killed after timeout", name)
	}

	instance.Process = nil
	instance.Stdin = nil
	instance.Stdout = nil
	instance.Stderr = nil

	log.Printf("Plugin %s stopped", name)
	return nil
}

// EnablePlugin enables a plugin
func (h *PluginHost) EnablePlugin(name string) error {
	// Validate plugin name to prevent path traversal
	if err := validatePluginName(name); err != nil {
		return fmt.Errorf("invalid plugin name: %w", err)
	}

	h.mu.Lock()
	instance, exists := h.plugins[name]
	if !exists {
		h.mu.Unlock()
		return fmt.Errorf("plugin %s not found", name)
	}
	instance.Enabled = true
	h.mu.Unlock()

	// Start plugin without holding the lock to avoid deadlock
	return h.StartPlugin(name)
}

// DisablePlugin disables a plugin
func (h *PluginHost) DisablePlugin(name string) error {
	// Validate plugin name to prevent path traversal
	if err := validatePluginName(name); err != nil {
		return fmt.Errorf("invalid plugin name: %w", err)
	}

	h.mu.Lock()
	instance, exists := h.plugins[name]
	if !exists {
		h.mu.Unlock()
		return fmt.Errorf("plugin %s not found", name)
	}
	instance.Enabled = false
	h.mu.Unlock()

	// Stop plugin without holding the lock to avoid deadlock
	return h.StopPlugin(name)
}

// GetPlugin returns a plugin instance
func (h *PluginHost) GetPlugin(name string) *PluginInstance {
	// Validate plugin name to prevent path traversal
	if err := validatePluginName(name); err != nil {
		return nil
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.plugins[name]
}

// UnloadPlugin removes a plugin from the host without deleting files
func (h *PluginHost) UnloadPlugin(name string) {
	// Validate plugin name to prevent path traversal
	if err := validatePluginName(name); err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.plugins, name)
}

// ListPlugins returns all loaded plugins
func (h *PluginHost) ListPlugins() map[string]*PluginInstance {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]*PluginInstance)
	for name, instance := range h.plugins {
		result[name] = instance
	}
	return result
}

// SendMessage sends a message to all enabled plugins
func (h *PluginHost) SendMessage(msg sdk.Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for name, instance := range h.plugins {
		if !instance.Enabled || instance.Process == nil {
			continue
		}

		req := sdk.PluginRequest{
			Type: "message",
			Data: mustMarshal(msg),
		}

		if err := h.sendRequest(instance, req); err != nil {
			log.Printf("Failed to send message to plugin %s: %v", name, err)
		}
	}
}

// ExecuteCommand executes a plugin command
func (h *PluginHost) ExecuteCommand(pluginName, command string, args []string) error {
	// Validate plugin name to prevent path traversal
	if err := validatePluginName(pluginName); err != nil {
		return fmt.Errorf("invalid plugin name: %w", err)
	}

	h.mu.RLock()
	instance, exists := h.plugins[pluginName]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	if !instance.Enabled || instance.Process == nil {
		return fmt.Errorf("plugin %s is not running", pluginName)
	}

	req := sdk.PluginRequest{
		Type:    "command",
		Command: command,
		Data:    mustMarshal(args),
	}

	return h.sendRequest(instance, req)
}

// UpdateUserList updates the list of online users
func (h *PluginHost) UpdateUserList(users []string) {
	h.mu.Lock()
	h.userList = users
	h.mu.Unlock()
}

// initializePlugin sends an initialization request to the plugin and waits for acknowledgment
func (h *PluginHost) initializePlugin(instance *PluginInstance) error {
	initData := map[string]interface{}{
		"config": instance.Config,
	}

	initRequest := sdk.PluginRequest{
		Type: "init",
		Data: mustMarshal(initData),
	}

	if err := h.sendRequest(instance, initRequest); err != nil {
		return fmt.Errorf("failed to send init request: %w", err)
	}

	instance.decoder = json.NewDecoder(instance.Stdout)

	type initResult struct {
		resp sdk.PluginResponse
		err  error
	}
	ch := make(chan initResult, 1)
	go func() {
		var resp sdk.PluginResponse
		if err := instance.decoder.Decode(&resp); err != nil {
			ch <- initResult{err: fmt.Errorf("failed to read init response: %w", err)}
			return
		}
		ch <- initResult{resp: resp}
	}()

	select {
	case result := <-ch:
		if result.err != nil {
			return result.err
		}
		if !result.resp.Success {
			errMsg := result.resp.Error
			if errMsg == "" {
				errMsg = "plugin returned unsuccessful init response"
			}
			return fmt.Errorf("plugin init failed: %s", errMsg)
		}
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("plugin did not respond to init within 5 seconds")
	}
}

// sendRequest sends a request to a plugin
func (h *PluginHost) sendRequest(instance *PluginInstance, req sdk.PluginRequest) error {
	if instance.Stdin == nil {
		return fmt.Errorf("plugin stdin is not available")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Sending request to plugin

	// Send request with newline delimiter
	data = append(data, '\n')
	_, err = instance.Stdin.Write(data)
	if err != nil {
		// Failed to write to plugin stdin
		return err
	}

	// Successfully sent request to plugin
	return nil
}

// handlePluginOutput handles stdout from a plugin
func (h *PluginHost) handlePluginOutput(instance *PluginInstance) {
	decoder := instance.decoder
	if decoder == nil {
		decoder = json.NewDecoder(instance.Stdout)
	}
	for {
		var response sdk.PluginResponse
		if err := decoder.Decode(&response); err != nil {
			if err == io.EOF {
				// Plugin stdout closed
				break
			}
			log.Printf("Failed to decode plugin %s response: %v", instance.Name, err)
			continue
		}

		// Received response from plugin
		h.handlePluginResponse(instance, response)
	}
}

// handlePluginErrors handles stderr from a plugin
func (h *PluginHost) handlePluginErrors(instance *PluginInstance) {
	scanner := json.NewDecoder(instance.Stderr)
	for {
		var logEntry struct {
			Level   string `json:"level"`
			Message string `json:"message"`
		}
		if err := scanner.Decode(&logEntry); err != nil {
			if err == io.EOF {
				break
			}
			break
		}
		log.Printf("[Plugin %s] %s: %s", instance.Name, logEntry.Level, logEntry.Message)
	}
}

// handlePluginResponse handles responses from plugins
func (h *PluginHost) handlePluginResponse(instance *PluginInstance, response sdk.PluginResponse) {
	switch response.Type {
	case "message":
		if response.Success {
			var msg sdk.Message
			if err := json.Unmarshal(response.Data, &msg); err != nil {
				log.Printf("Failed to unmarshal plugin message: %v", err)
				return
			}
			// Send message to chat
			select {
			case h.messageChan <- msg:
			default:
				log.Printf("Message channel full, dropping message from plugin %s", instance.Name)
			}
		}
	case "command_response":
		if response.Success {
			var text string
			if err := json.Unmarshal(response.Data, &text); err != nil {
				log.Printf("Failed to unmarshal command_response from plugin %s: %v", instance.Name, err)
				return
			}
			msg := sdk.Message{
				Sender:    instance.Name,
				Content:   text,
				CreatedAt: time.Now(),
				Type:      "text",
			}
			select {
			case h.messageChan <- msg:
			default:
				log.Printf("Message channel full, dropping command_response from plugin %s", instance.Name)
			}
		} else {
			log.Printf("Plugin %s command error: %s", instance.Name, response.Error)
		}
	case "log":
		if !response.Success {
			log.Printf("Plugin %s error: %s", instance.Name, response.Error)
		}
	}
}

// GetMessageChannel returns the channel for receiving messages from plugins
func (h *PluginHost) GetMessageChannel() <-chan sdk.Message {
	return h.messageChan
}

// mustMarshal is a helper function that panics on marshal error
func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal: %v", err))
	}
	return data
}
