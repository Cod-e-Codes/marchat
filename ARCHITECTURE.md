# Marchat Architecture Documentation

This document provides a comprehensive overview of the marchat system architecture, including component relationships, data flow, and design patterns.

## System Overview

Marchat is a self-hosted, terminal-based chat application built in Go with a client-server architecture. The system emphasizes security through end-to-end encryption, extensibility through a plugin system, and usability through multiple interface options.

## Core Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client TUI    │◄──►│  WebSocket      │◄──►│  Server Hub     │
│                 │    │  Communication  │    │                 │
│ • Chat Interface│    │ • JSON Messages │    │ • Message Hub   │
│ • File Transfer │    │ • E2E Encryption│    │ • User Mgmt     │
│ • Admin Panel   │    │ • Real-time     │    │ • Plugin Host   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Configuration   │    │ Shared Types    │    │ Database Layer  │
│ • Profiles      │    │ • Message Types │    │ • SQL Backends  │
│ • Encryption    │    │ • Crypto Utils  │    │ • Postgres/MySQL│
│ • Themes        │    │ • Protocols     │    │ • Dialect State │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Component Architecture

### Client Application (`client/`)

The client is a standalone terminal user interface built with the Bubble Tea framework. It's a complete application that can be built and run independently. The code is split across several files:

- **`main.go`**: Core model, state, Update loop, and command handlers
- **`cli_output.go`**: Lipgloss helpers for pre-TUI stdout (connection, E2E status, profile flow messages)
- **`hotkeys.go`**: Key binding definitions and methods
- **`render.go`**: Message rendering and UI display logic (optional per-line metadata: message id and encrypted flag)
- **`websocket.go`**: WebSocket connection management, send/receive, and E2E encryption helpers
- **`commands.go`**: Help text generation and command-related utilities
- **`notification_manager.go`**: Desktop/bell notification system

#### Core Models

- **`model`**: Main application state manager handling WebSocket communication, message rendering, and user interactions
- **`ConfigUIModel`**: Interactive configuration interface for server connection settings
- **`ProfileSelectionModel`**: Multi-profile management system for different server configurations (colored URL and `[Admin]` / `[E2E]` / `[Recent]` tags in the list)
- **`SensitiveDataModel`**: Secure credential input for admin keys and encryption passphrases (inactive fields use dim styling)
- **`codeSnippetModel`**: Code block rendering with syntax highlighting and selection capabilities
- **`filePickerModel`**: Interactive file selection interface with filtering and preview
- **`NotificationManager`**: Notification system supporting bell sounds and desktop notifications

#### Key Features

- Real-time chat with message history and user list
- Message editing, deletion, pinning, and reactions
- Direct messages between users
- Channel-based messaging (join/leave channels)
- Typing indicators (throttled, with timeout)
- Message search (server-side)
- Chat history export to file
- File sharing with configurable size limits (default 1MB), with optional E2E encryption
- Theme system supporting built-in and custom themes
- Administrative commands for user management
- End-to-end encryption with global key support (text and file transfers)
- Code snippet rendering with syntax highlighting
- Clipboard integration for text operations
- URL detection and external opening
- Tab completion for @mentions
- Connection status indicator
- Unread message count
- Multi-line input via Alt+Enter / Ctrl+J
- **Diagnostics**: `-doctor` and `-doctor-json` for environment, paths, and config checks (`internal/doctor`)

### Server Application (`cmd/server/main.go`)

The server is a standalone HTTP/WebSocket server application that provides real-time communication with plugin support and administrative interfaces. The ASCII banner is followed by **lipgloss**-styled status lines (WebSocket URL, admins, version, TLS state, tips, and optional admin-panel key hints) on stdout before the process settles into serving. Before serving, startup checks require at least one admin, a non-empty admin key, and a valid listen port; admin names are trimmed, lowercased, deduplicated case-insensitively, and must not be empty after trim.

#### Core Structures

- **`Hub`**: Central message routing system managing client connections, message broadcasting, channel management, and user state; tracks reserved usernames so handshake cannot double-book the same name before a client is registered
- **`Client`**: Individual WebSocket connection handler with read/write pumps and command processing
- **`AdminPanel`**: Terminal-based administrative interface for server management
- **`WebAdminServer`**: Web-based administrative interface with session authentication
- **`HealthChecker`**: System health monitoring with metrics collection
- **`PluginCommandHandler`**: Plugin command routing and execution system

#### Key Features

- Real-time message broadcasting to connected clients (channel-aware)
- Channel management: clients join/leave channels, messages routed per-channel
- Direct message routing between specific users
- Message editing, deletion, pinning, and search
- Typing indicator and read receipt broadcasting
- Reaction broadcasting
- User management including ban, kick, and allow operations
- Plugin command execution and management
- Database backup and maintenance operations (SQLite `VACUUM INTO` uses proper string quoting so backup paths containing `'` remain safe)
- System metrics collection and health monitoring
- Web-based admin panel with CSRF protection
- Health check endpoints for monitoring systems
- WebSocket message rate limiting
- **Diagnostics**: `-doctor` and `-doctor-json` without binding ports (`internal/doctor`)

### Server Library (`server/`)

The server package contains the core server logic and components that are used by the server application.

#### Core Components

- **WebSocket Handlers**: Connection management and message routing
- **Database Layer**: Pluggable SQL backends (SQLite/PostgreSQL/MySQL) with dialect-aware schema and query helpers
- **Admin Interfaces**: Both TUI and web-based administrative panels
- **Plugin Integration**: Plugin command handling and execution
- **Health Monitoring**: System metrics and health check endpoints

### Shared Components (`shared/`)

Common types and utilities used across client and server components.

#### Core Types

- **`Message`**: Standard chat message structure with encryption support, message IDs, recipient, channel, edited flag, and reaction metadata
- **`MessageType`**: Type discriminator (`text`, `file`, `admin_command`, `edit`, `delete`, `typing`, `reaction`, `dm`, `search`, `pin`, `read_receipt`, `join_channel`, `leave_channel`, `list_channels`)
- **`ReactionMeta`**: Emoji, target message ID, and removal flag
- **`EncryptedMessage`**: End-to-end encrypted message format (AEAD payload + metadata)
- **`Handshake`**: WebSocket connection authentication structure
- **`SessionKey`**: 32-byte ChaCha20-Poly1305 key material for global E2E (fingerprint in `KeyID` for logs)
- **`FileMeta`**: File transfer metadata including name, size, and data

#### Encryption System

Chat end-to-end encryption uses a **global symmetric key** shared by all clients (via `MARCHAT_GLOBAL_E2E_KEY` or manual distribution), not per-user key exchange on the wire.

- **ChaCha20-Poly1305**: Authenticated encryption for message and file confidentiality and integrity
- **Global E2E**: One 32-byte key per deployment; the client stores it in the passphrase-protected keystore (`EncryptMessage` / `DecryptMessage` in `shared`, orchestrated by `client/crypto/keystore.go`)
- **Keystore file**: Encrypted with **AES-GCM**; passphrase stretched with **PBKDF2** (see `client/crypto/keystore.go`)
- **File transfer**: Raw byte encryption/decryption via `EncryptRaw`/`DecryptRaw` in the keystore using the same global key

### Plugin System (`plugin/`)

Extensible architecture allowing custom functionality through external plugins.

#### Architecture

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

#### Components

- **SDK** (`plugin/sdk/`): Core plugin interface definitions and base implementations
- **Host** (`plugin/host/`): Subprocess management and JSON-based communication
- **Manager** (`plugin/manager/`): Plugin installation, store integration, and command execution
- **Store** (`plugin/store/`): Terminal-based plugin browsing and installation interface
- **License** (`plugin/license/`): Cryptographic license validation for official plugins (cached licenses re-verified on read)

#### Communication Protocol

Plugins communicate with the host through JSON messages over stdin/stdout:

- **Request Format**: `{"type": "message|command|init|shutdown", "command": "name", "data": {}}`
- **Response Format**: `{"type": "message|log", "success": true, "data": {}, "error": "message"}`
- **Message Types**: Initialization, message processing, command execution, graceful shutdown

### Configuration System

Flexible configuration management supporting multiple sources and interactive setup.

#### Server (`config/` Go package + runtime directory)

The **Go package** at repository path `config/` loads server settings from the process environment and from a `.env` file inside the **server configuration directory**. If `.env` exists, it is applied with **`godotenv.Overload`**: each `KEY=value` in the file **overwrites** that key in the process environment at startup (so the file is authoritative for keys it defines; keys set only in the environment and omitted from `.env` are unchanged). When `go.mod` is present in the process working directory, that directory defaults to `./config` in the repo (alongside the server’s `.env` and SQLite database). Otherwise it follows `MARCHAT_CONFIG_DIR` or the XDG-style user config path (see server `main` and `config` package). This `./config` folder is **not** where the TUI client stores `config.json` or profiles. See **README.md** (Configuration → Server `config/.env` vs process environment) and **deploy/CADDY-REVERSE-PROXY.md** (Breaking changes) for precedence details.

#### Client (`client/config/`)

The **client** stores `config.json`, `profiles.json`, keystore (unless legacy `keystore.dat` in cwd), themes, and debug logs under the **per-user application data directory** (e.g. `%APPDATA%\marchat` on Windows, `~/.config/marchat` on Linux), or under `MARCHAT_CONFIG_DIR` when set. This applies both when developing from a clone and when using release binaries.

#### Configuration Sources (server)

1. **Process environment**: Inherited `MARCHAT_*` and other variables before `.env` is read
2. **`.env` in server config directory**: Merged with **`godotenv.Overload`**; file entries override the same variable names already in the environment
3. **Interactive TUI**: User-friendly setup for initial configuration when required settings are missing
4. **Profile System** (client): Multiple server connection profiles in the client config directory

#### Key Settings (server-oriented)

- **Server Configuration**: Port, TLS certificates, admin authentication
- **Database Settings**: `MARCHAT_DB_PATH` selects backend and connection details (SQLite path or Postgres/MySQL DSN)
- **Plugin Configuration**: Registry URL and installation directories
- **Security Settings**: Admin keys, encryption keys, and authentication
- **File Transfer**: Size limits and allowed file types
- **Logging**: Log levels and output destinations

### Diagnostics (`internal/doctor`)

Shared package invoked by **`marchat-client`** and **`marchat-server`** when passed **`-doctor`** (human-readable report) or **`-doctor-json`** (JSON on stdout). It summarizes Go/OS, resolved config directories, known `MARCHAT_*` variables with secrets masked, role-specific checks (client: profiles, clipboard, TTY; server: `.env`, validation, detected DB dialect, DB connection-string format validation, DB/TLS ping checks), and optionally compares the embedded version to the latest GitHub release. For **server** doctor, the `MARCHAT_*` listing is captured **after** `LoadConfigWithoutValidation` applies **`godotenv.Overload`** on `config/.env`, matching effective runtime env; **client** doctor still reflects only the client process environment. The **text** report is **colorized** when stdout is a terminal and **`NO_COLOR`** is unset (otherwise plain); **`-doctor-json`** is always unstyled JSON. Set **`MARCHAT_DOCTOR_NO_NETWORK=1`** to skip the release check (e.g. air-gapped environments).

### Command Line Tools (`cmd/`)

Additional command-line utilities for system management and plugin licensing.

#### Available Commands

- **`cmd/server/main.go`**: Main server application with interactive configuration
- **`cmd/license/main.go`**: Plugin license management and validation tool

Both **`marchat-client`** and **`marchat-server`** embed diagnostics via **`internal/doctor`** (`-doctor`, `-doctor-json`); there is no separate doctor binary in release archives.

#### License Tool Features

- **License Generation**: Create signed licenses for official plugins
- **License Validation**: Verify plugin license authenticity
- **Key Management**: Generate Ed25519 key pairs for signing
- **Cache Management**: Offline license validation support; cached licenses are re-signature-checked (and `plugin_name` must match the cache key) on read

## Data Flow and Communication

### Message Flow

```
Client: Input → Encrypt → WebSocket Send
Server: WebSocket Receive → Hub → Plugin Processing → SQL Backend
Server: SQL Backend → Hub → WebSocket Broadcast  
Client: WebSocket Receive → Decrypt → Display
```

### Database Backends and Durability

- The server chooses a backend at runtime using `MARCHAT_DB_PATH`:
  - SQLite path (default local setup)
  - PostgreSQL DSN (`postgres://` / `postgresql://`)
  - MySQL DSN (`mysql:` / `mysql://`)
- Schema creation and upsert/insert-ignore SQL are dialect-aware.
- Placeholder rebinding keeps shared query callsites portable across backends.
- SQLite-specific optimizations (for example WAL mode) are applied only when the selected backend is SQLite.
- Durable state includes:
  - message history
  - reactions
  - read receipts
  - last channel per user

### WebSocket Protocol

The WebSocket communication uses JSON messages with the following structure:

- **Handshake**: Initial authentication with username and admin credentials
- **Messages**: Chat messages with optional encryption, message IDs, and channel/recipient metadata
- **Extended Types**: Edit, delete, typing, reaction, DM, search, pin, read receipt, join/leave/list channels
- **Commands**: Administrative and plugin commands
- **System Messages**: Connection status and user list updates

See [PROTOCOL.md](PROTOCOL.md) for the full message format specification.

### Encryption Flow

1. **Key setup**: Operator shares a 32-byte key (e.g. `openssl rand -base64 32` as `MARCHAT_GLOBAL_E2E_KEY`) or the first client generates one and operators copy it to peers
2. **Local storage**: Client holds the key in `keystore.dat` protected by `--keystore-passphrase`
3. **Message encryption**: Inner JSON payload is sealed with ChaCha20-Poly1305; nonce ‖ ciphertext is base64-encoded into `content` with `encrypted: true` (see **PROTOCOL.md**)
4. **Transport**: Standard WebSocket JSON; server does not decrypt
5. **Storage**: Server persists opaque ciphertext in the database

## Database Schema

### Tables

#### `messages`
```sql
CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id INTEGER DEFAULT 0,
    sender TEXT,
    content TEXT,
    created_at DATETIME,
    is_encrypted BOOLEAN DEFAULT 0,
    encrypted_data BLOB,
    nonce BLOB,
    recipient TEXT,
    edited BOOLEAN DEFAULT 0,
    deleted BOOLEAN DEFAULT 0,
    pinned BOOLEAN DEFAULT 0
);
```

#### `user_message_state`
```sql
CREATE TABLE user_message_state (
    username TEXT PRIMARY KEY,
    last_message_id INTEGER NOT NULL DEFAULT 0,
    last_seen DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

#### `ban_history`
```sql
CREATE TABLE ban_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    banned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    unbanned_at DATETIME,
    banned_by TEXT NOT NULL
);
```

### Key Features

- **Backend Selection**: `MARCHAT_DB_PATH` chooses SQLite/PostgreSQL/MySQL at runtime
- **WAL Mode (SQLite only)**: Write-Ahead Logging for better concurrency and crash recovery when SQLite is selected
- **SQLite Database Files**: `marchat.db` (main), `marchat.db-wal` (write-ahead log), `marchat.db-shm` (shared memory)
- **Message ID Tracking**: Sequential message IDs for user state management
- **Encryption Support**: Binary storage for encrypted message data
- **Performance Indexes**: Optimized queries for message retrieval and user state
- **Message Cap**: Automatic cleanup maintaining 1000 most recent messages
- **Ban History**: Comprehensive tracking of user moderation actions
- **Performance Tuning**: Backend-aware optimizations (SQLite pragmas when SQLite is selected)

## Administrative Interfaces

### Terminal Admin Panel

The TUI-based admin panel provides real-time server management:

- **System Monitoring**: Live metrics including connections, memory usage, and message rates
- **User Management**: Ban, kick, and allow operations with history tracking
- **Plugin Management**: Install, enable, disable, and uninstall plugins
- **Database Operations**: Backup, restore, and maintenance functions
- **System Controls**: Force garbage collection and metrics reset

Implementation notes (`server/admin_panel.go`):

- **Resize and layout**: Window size updates apply width and height through a single layout path so the help bar, user table, and plugin table stay in sync; content width is derived consistently for bordered panels.
- **Tabs**: Tab labels render at natural width with spacing instead of fixed equal-width cells, which reads better on narrow terminals.
- **Logs tab**: Log lines are ordered oldest-first so the newest entries appear at the bottom; when output exceeds the visible area, the view initially anchors to the bottom of the buffer.
- **Headings**: Section titles and info lines avoid embedding trailing newlines inside styled strings (newlines are written separately) so System and Metrics panels stay left-aligned.

### Web Admin Panel

The web-based interface provides the same functionality through a browser:

- **Session Authentication**: Secure login with admin key validation
- **CSRF Protection**: Cross-site request forgery prevention
- **Real-time Updates**: Live data refresh without page reloads
- **RESTful API**: Programmatic access to administrative functions
- **Responsive Design**: Works across desktop and mobile devices

## Security Architecture

### Authentication

- **Admin Key**: Shared secret for administrative access
- **Session Management**: Secure session tokens with expiration
- **CSRF Protection**: Token-based request validation
- **Rate Limiting**: Protection against brute force attacks

### Encryption

- **End-to-end (chat)**: Client-side ChaCha20-Poly1305 with a **shared global symmetric key**; the server never holds the plaintext key
- **Keystore**: PBKDF2-derived key from the keystore passphrase protects `keystore.dat` (AES-GCM)
- **Authenticated encryption**: ChaCha20-Poly1305 for chat and file payloads at the application layer

### Input Validation

- **Plugin Names**: Regex-based validation preventing path traversal
- **Username Validation**: Case-insensitive matching with length limits
- **File Upload**: Type and size validation with safe path handling
- **Command Parsing**: Structured command validation and sanitization

## Performance Considerations

### Concurrency

- **Goroutine-based**: Concurrent handling of WebSocket connections
- **Go Channel Communication**: Non-blocking message passing between components
- **Chat Channels**: Server-side channel management with per-channel broadcast routing
- **Connection Pooling**: Efficient database connection management
- **Plugin Isolation**: Separate processes prevent plugin crashes from affecting server

### Memory Management

- **Message Limits**: Automatic cleanup of old messages
- **Connection Tracking**: Efficient client state management
- **Plugin Lifecycle**: Proper cleanup of plugin subprocesses
- **Garbage Collection**: Configurable GC with monitoring

### Database Optimization

- **WAL Mode (SQLite only)**: Write-Ahead Logging enabled for improved concurrency and performance
- **Indexed Queries**: Performance indexes on frequently queried columns
- **Batch Operations**: Efficient bulk message operations
- **Connection Reuse**: Persistent database connections
- **Query Optimization**: Prepared statements for common operations
- **Performance Tuning**: SQLite-specific pragmas are applied only on SQLite; Postgres/MySQL use driver/backend defaults
- **Backup Considerations**: SQLite WAL mode creates additional files; backups may miss recent uncommitted data if taken while server is running

## Development Patterns

### Code Organization

- **Package-based**: Clear separation of concerns across packages
- **Interface-driven**: Dependency injection through interfaces
- **Model-View-Update**: Bubble Tea pattern for TUI components
- **Command Pattern**: Structured handling of administrative actions

### Error Handling

- **Graceful Degradation**: System continues operating despite component failures
- **Plugin Isolation**: Plugin failures don't affect core functionality
- **Comprehensive Logging**: Structured logging with component identification
- **User Feedback**: Clear error messages and status indicators

### Testing Strategy

- **Unit Tests**: Component-level testing with mock dependencies
- **Integration Tests**: End-to-end testing of client-server communication
- **Plugin Testing**: Isolated testing of plugin functionality
- **Performance Testing**: Load testing for concurrent connections

## Build and Deployment

### Application Structure

marchat produces two main executables:

- **`marchat-client`**: Built from `client/main.go` - the terminal chat client
- **`marchat-server`**: Built from `cmd/server/main.go` - the WebSocket server

### Build System

- **Cross-Platform Builds**: GitHub Actions (`.github/workflows/release.yml`) produces zips for **linux-amd64**, **linux-arm64**, **windows-amd64**, **darwin-amd64**, and **darwin-arm64**
- **Termux (Android aarch64)**: Use the **linux-arm64** release zip (static **`GOOS=linux`** binary); there is no separate `android-*` artifact
- **CGO**: Release CI sets **`CGO_ENABLED=0`** on the **`build`** job for static binaries and the pure Go SQLite stack, consistent with **`Dockerfile`**, **`build-release.ps1`**, and **`scripts/build-linux.sh`**
- **Release versioning**: A **`resolve-version`** job runs first and exports the tag (published release) or workflow input (`workflow_dispatch`); matrix jobs cannot expose outputs, so Docker image tags, release notes, and **`go build -ldflags`** all consume **`needs.resolve-version.outputs.version`**
- **GitHub release assets**: For **`release: published`** (not `workflow_dispatch`), **`upload-assets`** runs **`gh release upload`** for the artifact zips and the **`docker`** job appends the Docker Hub section with **`gh release edit`**, so publishing does not depend on **`softprops/action-gh-release`** (still **`node20`** in its `action.yml`, which triggers deprecation annotations if forced onto Node 24)
- **Release Scripts**: `build-release.ps1` and related helpers for local release-style builds
- **Docker Support**: Containerized deployment with health checks

### Configuration Management

- **Environment Variables**: Primary configuration method for containers
- **Interactive Setup**: User-friendly initial configuration through TUI
- **Profile Management**: Multiple server connection profiles on the client
- **Backward Compatibility**: Support for deprecated command-line flags
- **Diagnostics**: `-doctor` / `-doctor-json` on each binary for env/config verification (see `internal/doctor`)

### Container Support

- **Docker**: Official Docker images for easy deployment
- **Environment-based**: Configuration through environment variables
- **Volume Mounting**: Persistent data and configuration storage
- **Health Checks**: Built-in health monitoring for orchestration

### Cross-Platform Support

- **Operating Systems**: Linux, macOS, Windows, and Android/Termux (Termux: **linux/arm64** release asset)
- **Architecture Support**: AMD64 and ARM64 in official release zips
- **Terminal Compatibility**: Works with most terminal emulators
- **File System**: Handles different path separators and permissions

## Extension Points

### Plugin Development

- **SDK**: Comprehensive plugin development kit
- **Documentation**: Detailed plugin development guides
- **Examples**: Reference implementations for common patterns
- **Registry**: Centralized plugin distribution and discovery

### Custom Themes

- **JSON Configuration**: Declarative theme definition
- **Color Schemes**: Comprehensive color palette support
- **Component Styling**: Granular control over UI elements
- **Dynamic Loading**: Runtime theme switching

### Alternative Frontends

- **WebSocket Protocol**: Standardized communication layer enabling any frontend
- **JSON IPC**: Structured messages for easy parsing and integration
- **Encryption Support**: ChaCha20-Poly1305 (global symmetric E2E) for messaging and optional file encryption
- **Frontend Flexibility**: Architecture supports multiple frontend technologies
  - Web, desktop, or mobile clients can implement real-time chat, file transfer, and admin commands
- **Protocol Independence**: Frontends are decoupled from server implementation

### Administrative Extensions

- **Custom Commands**: Plugin-based command extensions
- **Webhooks**: External system integration
- **Metrics Export**: Custom monitoring and alerting
- **Backup Systems**: Custom backup and restore procedures
