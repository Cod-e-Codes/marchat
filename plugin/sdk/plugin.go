package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

// Plugin is the main interface that all marchat plugins must implement
type Plugin interface {
	// Name returns the plugin's unique identifier
	Name() string

	// Init initializes the plugin with configuration
	Init(config Config) error

	// OnMessage is called when a new message is received
	// Plugins can return additional messages to be sent
	OnMessage(msg Message) ([]Message, error)

	// Commands returns the list of commands this plugin registers
	Commands() []PluginCommand
}

// Config represents plugin configuration
type Config struct {
	PluginDir string            `json:"plugin_dir"`
	DataDir   string            `json:"data_dir"`
	Settings  map[string]string `json:"settings"`
}

// Message represents a chat message.
//
// Fields added after the initial SDK release (Channel, Encrypted, MessageID,
// Recipient, Edited) use omitempty and are backwards-compatible: older plugins
// that were compiled against the 4-field struct silently ignore the extra JSON
// keys on input and omit them on output.
type Message struct {
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Type      string    `json:"type,omitempty"`
	Channel   string    `json:"channel,omitempty"`
	Encrypted bool      `json:"encrypted,omitempty"`
	MessageID int64     `json:"message_id,omitempty"`
	Recipient string    `json:"recipient,omitempty"`
	Edited    bool      `json:"edited,omitempty"`
}

// PluginCommand represents a command that a plugin can register
type PluginCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
	AdminOnly   bool   `json:"admin_only"`
}

// PluginManifest contains metadata about a plugin
type PluginManifest struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	License     string            `json:"license"`
	Repository  string            `json:"repository,omitempty"`
	Homepage    string            `json:"homepage,omitempty"`
	Commands    []PluginCommand   `json:"commands"`
	Permissions []string          `json:"permissions"`
	Settings    map[string]string `json:"settings,omitempty"`
	MinVersion  string            `json:"min_version,omitempty"`
	MaxVersion  string            `json:"max_version,omitempty"`
}

// PluginResponse represents a response from a plugin
type PluginResponse struct {
	Type    string          `json:"type"`
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// PluginRequest represents a request to a plugin
type PluginRequest struct {
	Type    string          `json:"type"`
	Command string          `json:"command,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// PluginHost provides methods for plugins to interact with the host
type PluginHost interface {
	// SendMessage sends a message to the chat
	SendMessage(msg Message) error

	// GetUsers returns the list of online users
	GetUsers() []string

	// GetSetting retrieves a plugin setting
	GetSetting(key string) string

	// SetSetting stores a plugin setting
	SetSetting(key, value string) error

	// Log logs a message to the host's log system
	Log(level, message string)
}

// BasePlugin provides a basic implementation of the Plugin interface
type BasePlugin struct {
	name   string
	config Config
	host   PluginHost
}

// NewBasePlugin creates a new base plugin
func NewBasePlugin(name string) *BasePlugin {
	return &BasePlugin{
		name: name,
	}
}

// Name returns the plugin name
func (p *BasePlugin) Name() string {
	return p.name
}

// Init initializes the base plugin
func (p *BasePlugin) Init(config Config) error {
	p.config = config
	return nil
}

// OnMessage provides a default implementation that does nothing
func (p *BasePlugin) OnMessage(msg Message) ([]Message, error) {
	return nil, nil
}

// Commands provides a default implementation that returns no commands
func (p *BasePlugin) Commands() []PluginCommand {
	return nil
}

// SetHost sets the plugin host
func (p *BasePlugin) SetHost(host PluginHost) {
	p.host = host
}

// GetConfig returns the plugin configuration
func (p *BasePlugin) GetConfig() Config {
	return p.config
}

// GetHost returns the plugin host
func (p *BasePlugin) GetHost() PluginHost {
	return p.host
}

// ValidateManifest validates a plugin manifest
func ValidateManifest(manifest *PluginManifest) error {
	if manifest.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if manifest.Version == "" {
		return fmt.Errorf("plugin version is required")
	}
	if manifest.Description == "" {
		return fmt.Errorf("plugin description is required")
	}
	if manifest.Author == "" {
		return fmt.Errorf("plugin author is required")
	}
	if manifest.License == "" {
		return fmt.Errorf("plugin license is required")
	}
	return nil
}
