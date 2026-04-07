# marchat Plugin Ecosystem

This document provides a comprehensive overview of the plugin ecosystem implementation for marchat, covering architecture, development, and usage.

## Architecture Overview

The plugin ecosystem consists of several interconnected components:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Plugin SDK    │    │  Plugin Host    │    │ Plugin Manager  │
│                 │    │                 │    │                 │
│ • Core Interface│◄──►│ • Subprocess    │◄──►│ • Installation  │
│ • Communication │    │ • Lifecycle     │    │ • Store         │
│ • Base Classes  │    │ • JSON Protocol │    │ • Commands      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Plugin Store    │    │ License System  │    │ Command Handler │
│                 │    │                 │    │                 │
│ • TUI Interface │    │ • Validation    │    │ • Chat Commands │
│ • Registry      │    │ • Generation    │    │ • Integration   │
│ • Installation  │    │ • Caching       │    │ • Routing       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Core Components

### 1. Plugin SDK (`plugin/sdk/`)

**Purpose**: Provides the core interface and types for plugin development.

**Key Files**:
- `plugin.go` - Core Plugin interface and supporting types
- `BasePlugin` - Default implementation for common functionality

**Features**:
- Plugin interface with lifecycle methods
- Message processing and response system with extended context (channel, encryption status, message ID, recipient, edited flag)
- Command registration and execution
- Configuration management
- Manifest validation
- Backwards-compatible JSON wire format (new `omitempty` fields are silently ignored by older plugins)

### 2. Plugin Host (`plugin/host/`)

**Purpose**: Manages plugin subprocesses and communication.

**Key Files**:
- `host.go` - Plugin lifecycle and subprocess management

**Features**:
- Subprocess creation and management
- JSON communication over stdin/stdout
- Graceful shutdown with timeout
- Error handling and logging
- Message routing to plugins

### 3. Plugin Manager (`plugin/manager/`)

**Purpose**: High-level plugin management and installation.

**Key Files**:
- `manager.go` - Plugin installation, store integration, command execution

**Features**:
- Plugin installation from store
- Archive extraction (ZIP, TAR.GZ)
- Checksum validation
- Store integration
- Command execution

### 4. Plugin Store (`plugin/store/`)

**Purpose**: Terminal UI for browsing and installing plugins.

**Key Files**:
- `store.go` - Store interface and TUI implementation

**Features**:
- TUI-based plugin browsing
- Search and filtering
- One-click installation
- Plugin metadata display
- Offline cache support

### 5. License System (`plugin/license/`)

**Purpose**: Cryptographic license validation for official plugins.

**Key Files**:
- `validator.go` - License validation and generation

**Features**:
- Ed25519 signature validation
- License generation and caching
- Offline validation support
- Expiration checking
- On-disk cache integrity: each cache read re-checks `plugin_name` matches the requested plugin and re-verifies the signature; bad or mismatched cache files are removed

### 6. Command Integration (`server/`)

**Purpose**: Integrates plugin commands with the chat system.

**Key Files**:
- `plugin_commands.go` - Plugin command handling and routing

**Features**:
- Chat command integration
- Admin privilege checking
- Plugin message routing
- Command execution

### 7. License CLI (`cmd/license/`)

**Purpose**: Command-line tool for license management.

**Key Files**:
- `main.go` - License generation and validation CLI

**Features**:
- Key pair generation
- License generation
- License validation
- License status checking

## Plugin Communication Protocol

### Request Format
```json
{
  "type": "init|message|command|shutdown",
  "command": "command_name",
  "data": {}
}
```

### Response Format
```json
{
  "type": "message|log",
  "success": true,
  "data": {},
  "error": "error message"
}
```

### Message Data (type "message")

When the hub sends a `"message"` request, the `data` payload is an `sdk.Message` object:

```json
{
  "sender": "alice",
  "content": "hello world",
  "created_at": "2025-07-24T15:04:00Z",
  "type": "text",
  "channel": "general",
  "encrypted": false,
  "message_id": 42,
  "recipient": "",
  "edited": false
}
```

Zero-value fields (`channel` empty, `encrypted` false, `message_id` 0, etc.) are omitted from JSON via `omitempty`. Plugins compiled against older SDK versions silently ignore new keys.

### Request Types

1. **init**: Plugin initialization with configuration
2. **message**: Incoming chat message processing
3. **command**: Plugin command execution
4. **shutdown**: Graceful shutdown request

## Plugin Development

### Plugin Structure
```
myplugin/
├── plugin.json     # Plugin manifest
├── myplugin        # Binary executable
└── README.md       # Documentation
```

### Example Plugin Implementation
```go
type MyPlugin struct {
    *sdk.BasePlugin
}

func (p *MyPlugin) OnMessage(msg sdk.Message) ([]sdk.Message, error) {
    if msg.Encrypted {
        return nil, nil // content is opaque ciphertext
    }
    if strings.HasPrefix(msg.Content, "hello") {
        return []sdk.Message{{
            Sender:    "MyBot",
            Content:   "Hello back!",
            CreatedAt: time.Now(),
            Channel:   msg.Channel, // reply in the same channel
        }}, nil
    }
    return nil, nil
}

func (p *MyPlugin) Commands() []sdk.PluginCommand {
    return []sdk.PluginCommand{{
        Name:        "greet",
        Description: "Send a greeting",
        Usage:       ":greet <name>",
        AdminOnly:   false,
    }}
}
```

## Plugin Store Features

### TUI Interface
- Browse plugins by category, tags, or search
- View details including description, commands, metadata
- Install plugins with one-click installation
- Manage installed plugins (enable/disable/update)

### Registry Integration
- Community registry hosted on GitHub
- Offline caching for offline-first operation
- Automatic updates with `:refresh` command
- Checksum validation for security

## License System

### Official Plugin Licensing
- License files: `.license` files in plugin directories
- Cryptographic validation: Ed25519 signature verification
- Offline support: Licenses cached after first validation; cached copies are signature-checked on every read (and must match the plugin name in the cache filename)
- Expiration checking: Automatic license expiration handling

### License Management CLI
```bash
# Generate key pair
marchat-license -action genkey

# Generate license
marchat-license -action generate \
  -plugin myplugin \
  -customer CUSTOMER123 \
  -expires 2024-12-31 \
  -private-key <private-key>

# Validate license
marchat-license -action validate \
  -license myplugin.license \
  -public-key <public-key>
```

## Chat Integration

### Plugin Commands
- `:plugin list` - List installed plugins
- `:plugin enable <name>` - Enable a plugin
- `:plugin disable <name>` - Disable a plugin
- `:plugin uninstall <name>` - Uninstall a plugin (admin only)
- `:store` - Open plugin store
- `:refresh` - Refresh plugin store
- `:install <name>` - Install plugin from store

### Plugin Command Execution
- Dynamic routing: Commands routed to appropriate plugins
- Admin checking: Admin-only commands require privileges
- Error handling: Graceful error reporting
- Response integration: Plugin responses sent to chat

## Usage Examples

### Installing a Plugin
```bash
# Via chat command
:install echo

# Via plugin store
:store
# Navigate and select plugin, press Enter to install
```

### Using Plugin Commands
```bash
# Echo plugin command
:echo Hello, world!

# Weather plugin command
:weather New York

# Calculator plugin command
:calc 2 + 2 * 3
```

### Managing Plugins
```bash
# List installed plugins
:plugin list

# Enable a plugin
:plugin enable echo

# Disable a plugin
:plugin disable weather

# Uninstall a plugin (admin only)
:plugin uninstall calculator
```

## Configuration

### Plugin Directories
- Plugin directory: `./plugins/` (configurable)
- Data directory: `./plugin-data/` (configurable)
- Cache directory: `./plugin-cache/` (configurable)

Server-side paths are relative to the **server** process configuration (see [ARCHITECTURE.md](ARCHITECTURE.md)). For a quick server env check, run `marchat-server -doctor`.

### Environment Variables
```bash
MARCHAT_PLUGIN_DIR=./plugins
MARCHAT_PLUGIN_DATA_DIR=./plugin-data
MARCHAT_PLUGIN_REGISTRY_URL=https://raw.githubusercontent.com/Cod-e-Codes/marchat-plugins/main/registry.json
```

## Security Features

### Plugin Isolation
- Subprocess execution: Plugins run in isolated processes
- Resource limits: Automatic resource monitoring
- Graceful failure: Plugins cannot crash the main app
- Input validation: All plugin input validated

### License Security
- Cryptographic signatures: Ed25519 signature validation
- Offline validation: Licenses cached for offline use
- Expiration checking: Automatic license expiration handling
- Tamper detection: Signature verification on file load and on cached license reads; tampered or wrong-plugin cache entries are deleted

## Performance Considerations

### Optimization Features
- Async communication: Non-blocking plugin communication
- Resource monitoring: Automatic resource usage tracking
- Graceful shutdown: Timeout-based plugin termination
- Memory management: Efficient message routing

### Scalability
- Multiple plugins: Support for unlimited plugins
- Concurrent execution: Parallel plugin processing
- Message buffering: Efficient message queuing
- Cache optimization: Smart caching strategies

## Integration Points

### Server Integration
- Message routing: Automatic message forwarding to plugins
- Command handling: Dynamic command routing
- User list updates: Real-time user list synchronization
- Plugin lifecycle: Automatic plugin management

### Client Integration
- Command execution: Plugin commands via chat
- Store interface: TUI-based plugin browsing
- Status display: Plugin status in chat
- Error reporting: Plugin error messages in chat

## Testing and Validation

### Plugin Testing
- Unit tests: Individual plugin testing
- Integration tests: Plugin-host communication testing
- Performance tests: Resource usage validation
- Security tests: License validation testing

### Validation Features
- Manifest validation: Plugin.json format checking
- Binary validation: Executable file verification
- Checksum validation: Download integrity checking
- License validation: Cryptographic signature verification

## Future Enhancements

### Planned Features
- Plugin updates: Automatic plugin updating
- Dependency management: Plugin dependency resolution
- Advanced TUI: Enhanced store interface
- Plugin metrics: Usage and performance tracking
- Plugin marketplace: Enhanced discovery and distribution

### Community Features
- Plugin ratings: Community rating system
- Plugin reviews: User review system
- Plugin categories: Enhanced categorization
- Plugin search: Advanced search capabilities

## Design Principles

### Core Principles
1. Terminal-native: All interfaces optimized for terminal use
2. Offline-first: Works without internet connectivity
3. Modular: Clean separation of concerns
4. Secure: Cryptographic validation and isolation
5. Performant: Efficient resource usage and communication

### Architecture Benefits
- Extensibility: Easy to add new plugins
- Maintainability: Clean, modular code structure
- Reliability: Graceful error handling and recovery
- Security: Isolated execution and validation
- Usability: Intuitive command interface

## Documentation

### Developer Resources
- Plugin SDK: Complete API documentation
- Example plugins: Working plugin examples
- Best practices: Development guidelines
- Troubleshooting: Common issues and solutions

### User Resources
- Plugin commands: Complete command reference
- Store usage: Plugin store navigation guide
- License management: License validation guide
- Troubleshooting: User-facing issue resolution

This plugin ecosystem provides a comprehensive, secure, and user-friendly system for extending marchat's functionality while maintaining the terminal-native, offline-first design principles. 