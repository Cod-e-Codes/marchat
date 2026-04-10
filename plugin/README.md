# marchat Plugin System

The marchat plugin system provides a modular, extensible architecture for adding functionality to the chat application. Plugins are external binaries that communicate with marchat via JSON over stdin/stdout.

## Architecture Overview

### Plugin Communication
- Plugins run as **isolated subprocesses**
- Communication via **JSON over stdin/stdout**
- **Headless-first** design with optional TUI extensions
- **Graceful failure** - plugins cannot crash the main app

### Plugin Lifecycle
1. **Discovery**: Plugins are discovered in the plugin directory
2. **Loading**: Plugin manifest is parsed and validated
3. **Initialization**: Plugin receives configuration and user list
4. **Runtime**: Plugin processes messages and commands
5. **Shutdown**: Plugin receives shutdown signal and exits gracefully

## Plugin Structure

Each plugin must have the following structure:

```
myplugin/
├── plugin.json     # Plugin manifest
├── myplugin        # Binary executable
└── README.md       # Optional documentation
```

### Plugin Manifest (plugin.json)

```json
{
  "name": "myplugin",
  "version": "1.0.0",
  "description": "A description of what this plugin does",
  "author": "Your Name",
  "license": "MIT",
  "repository": "https://github.com/user/myplugin",
  "commands": [
    {
      "name": "mycommand",
      "description": "Description of the command",
      "usage": ":mycommand <args>",
      "admin_only": false
    }
  ],
  "permissions": [],
  "settings": {},
  "min_version": "0.1.0"
}
```

## Plugin SDK

### Core Interface

```go
type Plugin interface {
    Name() string
    Init(Config) error
    OnMessage(Message) ([]Message, error)
    Commands() []PluginCommand
}
```

### Message Type

The `sdk.Message` struct carries chat context from the hub to plugins and back:

```go
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
```

| Field | Description |
|-------|-------------|
| `Sender` | Username of the message author |
| `Content` | Message text (opaque ciphertext when `Encrypted` is true) |
| `CreatedAt` | Timestamp |
| `Type` | `"text"`, `"file"`, `"dm"`, etc. (matches `shared.MessageType` values) |
| `Channel` | Channel the message belongs to (empty = default `general`) |
| `Encrypted` | `true` when content is E2E encrypted; plugins should skip parsing |
| `MessageID` | Server-assigned ID (useful for reactions, edits) |
| `Recipient` | Target user for DMs (empty = broadcast) |
| `Edited` | `true` if the message was edited after send |

**Backwards compatibility**: All extended fields use `omitempty`. Plugins compiled against older SDK versions silently ignore unknown JSON keys and omit them on output; no recompile required.

**Message routing rules**:

- The hub only forwards messages with `type` set to `"text"` to plugins. Other types (typing, reactions, etc.) are not delivered.
- Plugin replies that **omit** `type` (or set it to anything other than `"text"`) are broadcast to clients but are **not** re-forwarded to other plugins. This prevents accidental infinite loops.
- To opt into **plugin-to-plugin chaining**, set `Type: "text"` on outbound `sdk.Message` explicitly. Use with care: the echo plugin, for example, should not do this or it will loop.
- **Encrypted messages**: The hub does not filter encrypted messages before forwarding to plugins. Plugins receive them with `Encrypted: true` and opaque `Content`. Plugins that parse `Content` should check `msg.Encrypted` and skip or handle accordingly.

### Message Processing

Plugins receive messages and can respond with additional messages:

```go
func (p *MyPlugin) OnMessage(msg sdk.Message) ([]sdk.Message, error) {
    // Skip encrypted messages the plugin cannot read
    if msg.Encrypted {
        return nil, nil
    }
    // Process incoming message
    if strings.HasPrefix(msg.Content, "hello") {
        response := sdk.Message{
            Sender:    "MyBot",
            Content:   "Hello back!",
            CreatedAt: time.Now(),
            Channel:   msg.Channel, // reply in the same channel
        }
        return []sdk.Message{response}, nil
    }
    return nil, nil
}
```

### Command Registration

Plugins can register commands that users can invoke:

```go
func (p *MyPlugin) Commands() []sdk.PluginCommand {
    return []sdk.PluginCommand{
        {
            Name:        "greet",
            Description: "Send a greeting",
            Usage:       ":greet <name>",
            AdminOnly:   false,
        },
    }
}
```

## Plugin Communication Protocol

### Request Format

```json
{
  "type": "message|command|init|shutdown",
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

### Message Types

- **init**: Plugin initialization with config and user list
- **message**: Incoming chat message
- **command**: Plugin command execution
- **shutdown**: Graceful shutdown request

## Plugin Development

### Getting Started

1. **Create plugin directory**:
   ```bash
   mkdir myplugin
   cd myplugin
   ```

2. **Create plugin.json**:
   ```json
   {
     "name": "myplugin",
     "version": "1.0.0",
     "description": "My first plugin",
     "author": "Your Name",
     "license": "MIT",
     "commands": []
   }
   ```

3. **Create main.go**:
   ```go
   package main

   import (
       "encoding/json"
       "fmt"
       "os"
       "time"

       "github.com/Cod-e-Codes/marchat/plugin/sdk"
   )

   type MyPlugin struct {
       *sdk.BasePlugin
   }

   func NewMyPlugin() *MyPlugin {
       return &MyPlugin{
           BasePlugin: sdk.NewBasePlugin("myplugin"),
       }
   }

   func (p *MyPlugin) Init(config sdk.Config) error {
       return nil
   }

   func (p *MyPlugin) OnMessage(msg sdk.Message) ([]sdk.Message, error) {
       return nil, nil
   }

   func (p *MyPlugin) Commands() []sdk.PluginCommand {
       return nil
   }

   func main() {
       plugin := NewMyPlugin()
       
       decoder := json.NewDecoder(os.Stdin)
       encoder := json.NewEncoder(os.Stdout)
       
       for {
           var req sdk.PluginRequest
           if err := decoder.Decode(&req); err != nil {
               break
           }
           
           response := handleRequest(plugin, req)
           encoder.Encode(response)
       }
   }

   func handleRequest(plugin *MyPlugin, req sdk.PluginRequest) sdk.PluginResponse {
       // Handle different request types
       switch req.Type {
       case "init":
           // Handle initialization
       case "message":
           // Handle incoming message
       case "command":
           // Handle command execution
       case "shutdown":
           // Handle shutdown
       }
       
       return sdk.PluginResponse{
           Type:    req.Type,
           Success: true,
       }
   }
   ```

4. **Build the plugin**:
   ```bash
   go build -o myplugin main.go
   ```

5. **Install the plugin**:
   ```bash
   # Copy to plugin directory
   cp myplugin /path/to/marchat/plugins/myplugin/
   cp plugin.json /path/to/marchat/plugins/myplugin/
   ```

### Plugin Configuration

Plugins receive configuration during initialization:

```go
type Config struct {
    PluginDir string            // Plugin directory path
    DataDir   string            // Plugin data directory
    Settings  map[string]string // Plugin settings
}
```

### Plugin Data Storage

Plugins can store data in their data directory:

```go
func (p *MyPlugin) saveData(data interface{}) error {
    dataFile := filepath.Join(p.config.DataDir, "data.json")
    return os.WriteFile(dataFile, data, 0644)
}
```

## Plugin Management

### Installation

Plugins can be installed via:

1. **Chat commands**:
   ```
   :install myplugin
   ```

2. **Plugin store**:
   ```
   :store
   ```

3. **Manual installation**:
   - Copy plugin files to plugin directory
   - Restart marchat or use `:plugin enable myplugin`

### Plugin Commands

- `:plugin list` - List installed plugins
- `:plugin enable <name>` - Enable a plugin
- `:plugin disable <name>` - Disable a plugin
- `:plugin uninstall <name>` - Uninstall a plugin (admin only)
- `:store` - Open plugin store
- `:refresh` - Refresh plugin store

### Plugin Store

The plugin store provides a TUI interface for browsing and installing plugins:

- **Browse plugins** by category, tags, or search
- **View plugin details** including description, commands, and metadata
- **Install plugins** with one-click installation
- **Manage installed plugins** enable/disable/update

## Official Plugins and Licensing

### License Validation

Official (paid) plugins require license validation:

1. **License file**: `.license` file in plugin directory
2. **Cryptographic verification**: Ed25519 signature validation
3. **Offline support**: Licenses cached after first validation; the server re-verifies the signature (and that `plugin_name` matches the cache key) whenever it reads the cache, and drops invalid cache files

### License Management

Use the `marchat-license` CLI tool:

```bash
# Generate key pair
marchat-license -action genkey

# Generate license
marchat-license -action generate \
  -plugin myplugin \
  -customer CUSTOMER123 \
  -expires 2024-12-31 \
  -private-key <private-key> \
  -output myplugin.license

# Validate license
marchat-license -action validate \
  -license myplugin.license \
  -public-key <public-key>

# Check license status
marchat-license -action check \
  -plugin myplugin \
  -public-key <public-key>
```

## Community Plugin Registry

### Registry Format

The community registry is a JSON file hosted on GitHub:

```json
[
  {
    "name": "myplugin",
    "version": "1.0.0",
    "description": "A community plugin",
    "author": "Community Member",
    "license": "MIT",
    "repository": "https://github.com/user/myplugin",
    "download_url": "https://github.com/user/myplugin/releases/latest/download/myplugin.zip",
    "checksum": "sha256:...",
    "category": "utility",
    "tags": ["chat", "utility"],
    "commands": [...]
  }
]
```

### Submitting Plugins

1. **Create plugin** following the structure above
2. **Host plugin** on GitHub/GitLab with releases
3. **Submit PR** to the community registry
4. **Include metadata** in registry entry

### Registry URL

The default registry URL is:
```
https://raw.githubusercontent.com/Cod-e-Codes/marchat-plugins/main/registry.json
```

## Best Practices

### Plugin Development

1. **Fail gracefully**: Never crash the main application
2. **Use BasePlugin**: Extend `sdk.BasePlugin` for common functionality
3. **Validate input**: Always validate user input and plugin data
4. **Log appropriately**: Use stderr for logging, stdout for responses
5. **Handle errors**: Return meaningful error messages
6. **Test thoroughly**: Test with various inputs and edge cases

### Security Considerations

1. **Input validation**: Validate all user input
2. **Resource limits**: Don't consume excessive resources
3. **File operations**: Use plugin data directory for file operations
4. **Network access**: Document any network access requirements
5. **Permissions**: Request only necessary permissions

### Performance Guidelines

1. **Async operations**: Use goroutines for long-running operations
2. **Memory usage**: Be mindful of memory consumption
3. **Response time**: Respond quickly to avoid blocking the chat
4. **Caching**: Cache frequently accessed data
5. **Cleanup**: Clean up resources on shutdown

## Example Plugins

### Echo Plugin

A simple echo plugin that repeats messages:

```go
func (p *EchoPlugin) OnMessage(msg sdk.Message) ([]sdk.Message, error) {
    if strings.HasPrefix(msg.Content, "echo:") {
        response := sdk.Message{
            Sender:    "EchoBot",
            Content:   strings.TrimPrefix(msg.Content, "echo:"),
            CreatedAt: time.Now(),
        }
        return []sdk.Message{response}, nil
    }
    return nil, nil
}
```

### Weather Plugin

A weather plugin that responds to weather queries:

```go
func (p *WeatherPlugin) OnMessage(msg sdk.Message) ([]sdk.Message, error) {
    if strings.HasPrefix(msg.Content, "weather:") {
        location := strings.TrimPrefix(msg.Content, "weather:")
        weather := p.getWeather(location)
        
        response := sdk.Message{
            Sender:    "WeatherBot",
            Content:   fmt.Sprintf("Weather in %s: %s", location, weather),
            CreatedAt: time.Now(),
        }
        return []sdk.Message{response}, nil
    }
    return nil, nil
}
```

## Troubleshooting

### Common Issues

1. **Plugin not loading**: Check plugin.json format and binary permissions
2. **Plugin not responding**: Check JSON communication format
3. **Permission denied**: Ensure plugin binary is executable
4. **License validation failed**: Check license file and public key
5. **Plugin crashes**: Check plugin logs in stderr

### Debugging

1. **Enable debug logging**: Set log level to debug
2. **Check plugin logs**: Plugin stderr is logged by marchat
3. **Test communication**: Use test harness for plugin communication
4. **Validate JSON**: Ensure JSON format is correct
5. **Check permissions**: Verify file and directory permissions

### Getting Help

- **Documentation**: Check this README and code comments
- **Examples**: Review example plugins in `plugin/examples/`
- **Issues**: Report bugs on GitHub
- **Discussions**: Ask questions in GitHub Discussions
- **Community**: Join the marchat community

## License

The plugin system is part of marchat and is licensed under the MIT License. Individual plugins may have their own licenses. 