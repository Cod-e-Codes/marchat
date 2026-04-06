# marchat

<img src="assets/marchat-transparent.svg" alt="marchat - terminal chat application" width="200" height="auto">

[![Go CI](https://github.com/Cod-e-Codes/marchat/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/Cod-e-Codes/marchat/actions/workflows/go.yml)
[![MIT License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Cod-e-Codes/marchat?logo=go)](https://go.dev/dl/)
[![GitHub all releases](https://img.shields.io/github/downloads/Cod-e-Codes/marchat/total?logo=github)](https://github.com/Cod-e-Codes/marchat/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/codecodesxyz/marchat?logo=docker)](https://hub.docker.com/r/codecodesxyz/marchat)
[![Version](https://img.shields.io/badge/version-v0.11.0--beta.2-blue)](https://github.com/Cod-e-Codes/marchat/releases/tag/v0.11.0-beta.2)

A lightweight terminal chat with real-time messaging over WebSockets, optional E2E encryption, and a flexible plugin ecosystem. Built for developers who prefer the command line.

**Quick start:** [QUICKSTART.md](QUICKSTART.md) for a single-page walkthrough (install → server → client → next docs).

## Latest Updates

### v0.11.0-beta.2 (Current)
- **Go 1.25.8** across CI, Docker, and docs; **SECURITY.md** updates (supported versions, edwards25519 note)
- **UX**: Terminal-native chrome (reaction/message emoji unchanged); **Alt+M** / **`:msginfo`** toggle message metadata; colorized server banner and client pre-TUI (**`NO_COLOR`** respected)
- **Doctor**: TTY color for text mode; server **`MARCHAT_*`** reflects **`config/.env`**; docs for **`-doctor-json`** / **`NO_COLOR`**
- **Server**: Hardened license cache, username reservation, DB backup SQL; **CI**: static release builds; published releases attach zips with **`gh release upload`** and append the Docker blurb with **`gh release edit`** (avoids Node 20–labeled JS actions); **Termux** → **linux-arm64** assets

### Earlier
- **v0.11.0-beta.1**: **[PR #83](https://github.com/Cod-e-Codes/marchat/pull/83)**: SQLite / PostgreSQL / MySQL, durable reactions & read receipts, message-state layer; release **`resolve-version`** + static builds; serialized WS writes; admin TUI & doctor DB checks (see **ARCHITECTURE.md**, **PROTOCOL.md**)
- **v0.10.0-beta.3**: Caddy TLS proxy sample ([**deploy/CADDY-REVERSE-PROXY.md**](deploy/CADDY-REVERSE-PROXY.md)), client WSS/TLS & direct-connect UX, **`config/.env`** precedence docs
- **v0.10.0-beta.2**: **`-doctor`** / **`-doctor-json`**, **`CGO_ENABLED=0`** builds, sqlite bump, Docker entrypoint & volume permissions
- **v0.10.0-beta.1**: Edit/delete/pin/search, reactions, DMs, channels, typing, E2E files, plugins, rate limits, Docker Compose; client modularization

Full changelog on [GitHub releases](https://github.com/Cod-e-Codes/marchat/releases).

## Demos

Screen recordings of a current build (GIF autoplay depends on the viewer).

### Server: startup banner and web admin panel

![Server startup and web admin panel](assets/demo-server-admin-panel.gif "marchat server startup and web admin panel")

### Server: diagnostics (`marchat-server -doctor`)

![Server diagnostics (doctor)](assets/demo-server-doctor.gif "marchat-server -doctor output")

### Client: reactions and help

![Client reactions and help](assets/demo-client-reactions-help.gif "marchat client reactions and help")

### Client: theme switching (`:theme`)

![Client theme switching](assets/demo-client-theme-switch.gif "marchat client :theme")

### Client: diagnostics (`marchat-client -doctor`)

![Client diagnostics (doctor)](assets/demo-client-doctor.gif "marchat-client -doctor output")

## Features

- **Terminal UI** - Beautiful TUI built with Bubble Tea
- **Real-time Chat** - Fast WebSocket messaging with SQLite, PostgreSQL, or MySQL backends
- **Message Management** - Edit, delete, pin, react to, and search messages
- **Direct Messages** - Private DM conversations between users
- **Channels** - Multiple chat rooms with join/leave and per-channel messaging
- **Typing Indicators** - See when other users are typing
- **Read Receipts** - Message read acknowledgement (broadcast-level)
- **Plugin System** - Remote registry with text commands and Alt+key hotkeys
- **E2E Encryption** - X25519/ChaCha20-Poly1305 with global encryption, including file transfers
- **File Sharing** - Send files up to 1MB (configurable) with interactive picker and optional E2E encryption
- **Admin Controls** - User management, bans, kick system with ban history gaps
- **Smart Notifications** - Bell + desktop notifications with quiet hours and focus mode ([guide](NOTIFICATIONS.md))
- **Themes** - Built-in themes + custom themes via JSON ([guide](THEMES.md))
- **Docker Support** - Containerized deployment with `docker-compose.yml` for local dev; optional **TLS reverse proxy** via Caddy ([guide](deploy/CADDY-REVERSE-PROXY.md))
- **Health Monitoring** - `/health` and `/health/simple` endpoints with system metrics
- **Structured Logging** - JSON logs with component separation and user tracking
- **UX Enhancements** - Connection status indicator, tab completion for @mentions, unread message count, multi-line input, chat export
- **Cross-Platform** - Runs on Linux, macOS, Windows, and Android/Termux
- **Diagnostics** - `marchat-client -doctor` and `marchat-server -doctor` (or `-doctor-json`) summarize environment, resolved paths, and configuration health

## Overview

marchat started as a fun weekend project for father-son coding sessions and has evolved into a lightweight, self-hosted terminal chat application designed specifically for developers who love the command line. It supports SQLite by default and can also run against PostgreSQL or MySQL for larger deployments.

**Key Benefits:**
- **Self-hosted**: No external services required
- **Cross-platform**: Linux, macOS, Windows, and Android/Termux
- **Secure**: Optional E2E encryption with X25519/ChaCha20-Poly1305
- **Extensible**: Plugin ecosystem for custom functionality
- **Lightweight**: Minimal resource usage, perfect for servers

## Quick Start

### 1. Generate Admin Key
```bash
openssl rand -hex 32
```

### 2. Start Server

**Option A: Environment Variables (Recommended)**
```bash
export MARCHAT_ADMIN_KEY="your-generated-key"
export MARCHAT_USERS="admin1,admin2"
./marchat-server

# With admin panel
./marchat-server --admin-panel

# With web panel
./marchat-server --web-panel
```

**Option B: Interactive setup (first run, missing required config)**
```bash
./marchat-server --interactive
```
Runs a guided wizard **only when** `MARCHAT_ADMIN_KEY` or `MARCHAT_USERS` is not set. If they are already in the environment or `config/.env`, the server starts normally and `--interactive` does nothing extra.

### 3. Connect Client
```bash
# Admin connection
./marchat-client --username admin1 --admin --admin-key your-key --server ws://localhost:8080/ws

# Regular user
./marchat-client --username user1 --server ws://localhost:8080/ws

# Or use interactive mode
./marchat-client
```

## Database Schema

Key tables for message tracking and moderation:
- **messages**: Core message storage with `message_id`
- **user_message_state**: Per-user message history state
- **ban_history**: Ban/unban event tracking for history gaps

## Installation

**Binary Installation:**
```bash
# Linux (amd64)
wget https://github.com/Cod-e-Codes/marchat/releases/download/v0.11.0-beta.2/marchat-v0.11.0-beta.2-linux-amd64.zip
unzip marchat-v0.11.0-beta.2-linux-amd64.zip && chmod +x marchat-*

# macOS (amd64)
wget https://github.com/Cod-e-Codes/marchat/releases/download/v0.11.0-beta.2/marchat-v0.11.0-beta.2-darwin-amd64.zip
unzip marchat-v0.11.0-beta.2-darwin-amd64.zip && chmod +x marchat-*

# Windows - PowerShell
iwr -useb https://raw.githubusercontent.com/Cod-e-Codes/marchat/main/install.ps1 | iex
```

**Docker:**
```bash
docker pull codecodesxyz/marchat:v0.11.0-beta.2
docker run -d -p 8080:8080 \
  -e MARCHAT_ADMIN_KEY=$(openssl rand -hex 32) \
  -e MARCHAT_USERS=admin1,admin2 \
  codecodesxyz/marchat:v0.11.0-beta.2
```

**Docker Compose (local development):**

The `server` service loads **`config/.env` first, then a project-root `.env`** (both optional and gitignored). Put **`MARCHAT_ADMIN_KEY`** and **`MARCHAT_USERS`** in either file (see [Essential Environment Variables](#essential-environment-variables)). Compose also sets **`MARCHAT_DB_PATH=/data/marchat.db`** so SQLite uses the attached volume.

Example snippet for `config/.env` or `.env` (generate a strong key for anything reachable from a network):

```bash
MARCHAT_ADMIN_KEY=your-secret-here
MARCHAT_USERS=admin1,admin2
```

Then:

```bash
docker compose up -d
```

**TLS reverse proxy (Caddy, optional):** To terminate TLS in front of a **host-native** `marchat-server` (plain HTTP on port 8080), use `docker-compose.proxy.yml`, `deploy/caddy/Caddyfile`, and `deploy/caddy/proxy.env.example` plus optional gitignored `deploy/caddy/proxy.env` for **`MARCHAT_CADDY_EXTRA_HOSTS`** (public IP/DNS on `tls internal`). Published port **8443** maps to HTTPS/WebSocket inside the container; clients use `wss://localhost:8443/ws` (with `--skip-tls-verify` while using Caddy’s internal CA). The proxy stack must be running whenever you use that URL. Full steps, helper scripts (`scripts/build-windows.ps1` / `scripts/build-linux.sh`, `scripts/connect-local-wss.ps1` / `scripts/connect-local-wss.sh`), source changes, and breaking notes: **[deploy/CADDY-REVERSE-PROXY.md](deploy/CADDY-REVERSE-PROXY.md)**.

**From Source:**
```bash
git clone https://github.com/Cod-e-Codes/marchat.git && cd marchat
go mod tidy
go build -o marchat-server ./cmd/server
go build -o marchat-client ./client
```

**Prerequisites for source build:**
- Go 1.25.8 or later ([download](https://go.dev/dl/))
- Linux clipboard support: `sudo apt install xclip` (Ubuntu/Debian) or `sudo yum install xclip` (RHEL/CentOS)

**Terminal colors:** The server startup banner and the client’s pre-chat output (connection, E2E status, profile picker tags such as `[Admin]` / `[E2E]`, and auth prompts) use [lipgloss](https://github.com/charmbracelet/lipgloss) for emphasis. Set **`NO_COLOR=1`** (or **`NO_COLOR`**) in the environment to disable colors on plain stdout/stderr.

## Configuration

### Essential Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MARCHAT_ADMIN_KEY` | Yes | - | Admin authentication key |
| `MARCHAT_USERS` | Yes | - | Comma-separated admin usernames |
| `MARCHAT_PORT` | No | `8080` | Server port |
| `MARCHAT_DB_PATH` | No | `./config/marchat.db` | Database path/DSN. Supports SQLite file path, `postgres://...`, or `mysql:...` |
| `MARCHAT_TLS_CERT_FILE` | No | - | TLS certificate (enables wss://) |
| `MARCHAT_TLS_KEY_FILE` | No | - | TLS private key |
| `MARCHAT_GLOBAL_E2E_KEY` | No | - | Base64 32-byte global encryption key |
| `MARCHAT_MAX_FILE_BYTES` | No | `1048576` | Max file size in bytes (1MB default) |
| `MARCHAT_MAX_FILE_MB` | No | `1` | Max file size in MB (alternative to bytes) |
| `MARCHAT_ALLOWED_USERS` | No | - | Username allowlist (comma-separated) |

**Additional variables:** `MARCHAT_LOG_LEVEL`, `MARCHAT_CONFIG_DIR`, `MARCHAT_BAN_HISTORY_GAPS`, `MARCHAT_PLUGIN_REGISTRY_URL`

### Database backend setup

`MARCHAT_DB_PATH` accepts either a SQLite path or a DSN-style backend URL/prefix:

```bash
# SQLite (default)
export MARCHAT_DB_PATH=./config/marchat.db

# PostgreSQL
export MARCHAT_DB_PATH='postgres://marchat:marchat@127.0.0.1:5432/marchat?sslmode=disable'

# MySQL
export MARCHAT_DB_PATH='mysql:marchat:marchat@tcp(127.0.0.1:3306)/marchat?parseTime=true'
```

Notes:
- PostgreSQL requires a reachable database and credentials with schema/table create permissions.
- MySQL DSNs should include `parseTime=true` so timestamp fields decode correctly.
- **MariaDB** generally works with the same `mysql:` DSN shape and `parseTime=true` as MySQL.
- The server creates **dialect-specific DDL** (for example, MySQL/MariaDB use fixed-width strings where indexes, primary keys, or unique constraints apply, because full `TEXT` keys are rejected). Long message bodies still use a large text type.
- SQLite remains the easiest local development option.

**Doctor / diagnostics:** Set `MARCHAT_DOCTOR_NO_NETWORK` to `1` to skip the GitHub latest-release check in `-doctor` / `-doctor-json`.

**File Size Configuration:** Use either `MARCHAT_MAX_FILE_BYTES` (exact bytes) or `MARCHAT_MAX_FILE_MB` (megabytes). If both are set, `MARCHAT_MAX_FILE_BYTES` takes priority.

**Interactive Setup:** Use `--interactive` flag for guided server configuration when environment variables are missing.

### Server: `config/.env` vs process environment

The server loads **`{config directory}/.env`** (for a repo clone, usually **`config/.env`**) if the file exists, using **`godotenv.Overload`**.

| Situation | Effect |
|-----------|--------|
| A variable appears **in `.env`** | That value **replaces** the same name already in the process environment when the server starts. |
| A variable is set **only** in the environment (not in `.env`) | It is **unchanged** by `.env` loading. |
| No `.env` file | Configuration comes only from the environment, flags, and defaults. |

**Why:** Older `godotenv.Load` behavior skipped keys already set in the environment, so a stale shell `MARCHAT_ADMIN_KEY` could override an updated `config/.env`. `Overload` makes the file authoritative for any key it defines.

**Operational notes:** Restart the server after editing `.env`. If you deploy with both injected secrets and a mounted `.env`, any **overlapping** key in the file wins at startup. See **[deploy/CADDY-REVERSE-PROXY.md](deploy/CADDY-REVERSE-PROXY.md#breaking-changes)** for migration and edge cases.

**Not the same as Docker Compose’s `.env`:** Compose’s file next to `docker-compose.yml` is for **substituting** `${VAR}` into YAML; the table above is about **marchat-server** reading **`config/.env`** at runtime.

### Client vs server config locations

| Role | Default location | Override |
|------|------------------|----------|
| **Server** (`.env`, SQLite DB, debug log) | In development from a repo clone: `./config` next to `go.mod`. Otherwise `MARCHAT_CONFIG_DIR` or the user config path (see [ARCHITECTURE.md](ARCHITECTURE.md)). | `MARCHAT_CONFIG_DIR`, `--config-dir` |
| **Client** (`config.json`, `profiles.json`, keystore, `themes.json`) | Per-user app data (e.g. Windows `%APPDATA%\marchat`, Linux/macOS `~/.config/marchat`). Same when developing from source. | `MARCHAT_CONFIG_DIR` |

The repository’s `config/` directory holds **server** runtime files and the **Go package** `github.com/Cod-e-Codes/marchat/config`; it is not the client’s profile folder.

### Diagnostics (`-doctor`)

Run **`./marchat-client -doctor`** or **`./marchat-server -doctor`** for a text report (paths, masked `MARCHAT_*` env, sanity checks). **Server** doctor lists `MARCHAT_*` **after** loading the resolved config directory’s **`.env`** (same as the running server), so values are not limited to what your shell exported. **Client** doctor only shows variables present in the client process (it does not read the server’s `config/.env`). Server doctor also reports the detected DB dialect, validates the configured DB connection string format, and attempts a DB ping. On a **color-capable terminal** (stdout is a TTY), the text report uses **ANSI colors** aligned with the server pre-TUI banner; set **`NO_COLOR`** or redirect to a file/pipe for **plain** output. Use **`-doctor-json`** for machine-readable output (never colorized). If both flags were passed, `-doctor-json` wins. Exits without starting the TUI or listening on a port. See [ARCHITECTURE.md](ARCHITECTURE.md) for details.

## Admin Commands

### User Management
| Command | Description | Hotkey |
|---------|-------------|--------|
| `:ban <user>` | Permanent ban | `Ctrl+B` (with user selected) |
| `:kick <user>` | 24h temporary ban | `Ctrl+K` (with user selected) |
| `:unban <user>` | Remove permanent ban | `Ctrl+Shift+B` |
| `:allow <user>` | Override kick early | `Ctrl+Shift+A` |
| `:forcedisconnect <user>` | Force disconnect user | `Ctrl+F` (with user selected) |
| `:cleanup` | Clean stale connections | - |

### Database Operations (`:cleardb` or `Ctrl+D` menu)
- **Clear DB** - Wipe all messages
- **Backup DB** - Create database backup
- **Show Stats** - Display database statistics

## User Commands

### General
| Command | Description | Hotkey |
|---------|-------------|--------|
| `:theme <name>` | Switch theme (built-in or custom) | `Ctrl+T` (cycles) |
| `:themes` | List all available themes | - |
| `:time` | Toggle 12/24-hour format | `Alt+T` |
| `:msginfo` | Toggle message metadata (message id / encrypted) on chat lines | `Alt+M` |
| `:clear` | Clear chat buffer | `Ctrl+L` |
| `:q` | Quit application (vim-style) | - |
| `:sendfile [path]` | Send file (or open picker without path) | `Alt+F` |
| `:savefile <name>` | Save received file | - |
| `:code` | Open code composer with syntax highlighting | `Alt+C` |
| `:export [file]` | Export chat history to a text file | - |

### Messaging
| Command | Description |
|---------|-------------|
| `:edit <id> <text>` | Edit a message by its ID |
| `:delete <id>` | Delete a message by its ID |
| `:dm [user] [msg]` | Send a DM or toggle DM mode (no args exits DM mode) |
| `:search <query>` | Search message history on the server |
| `:react <id> <emoji>` | React to a message (supports aliases: `+1`, `heart`, `fire`, `party`, `laugh`, `eyes`, `check`, `rocket`, `think`, etc.) |
| `:pin <id>` | Toggle pin on a message |
| `:pinned` | List all pinned messages |

### Channels
| Command | Description |
|---------|-------------|
| `:join <channel>` | Join a channel (clients start in `#general`) |
| `:leave` | Leave current channel, return to `#general` |
| `:channels` | List active channels with user counts |

### Notifications
| Command | Description | Hotkey |
|---------|-------------|--------|
| `:notify-mode <mode>` | Set notification mode (none/bell/desktop/both) | `Alt+N` (toggle desktop) |
| `:bell` | Toggle bell notifications | - |
| `:bell-mention` | Toggle mention-only notifications | - |
| `:focus [duration]` | Enable focus mode (mute notifications) | - |
| `:quiet <start> <end>` | Set quiet hours (e.g., `:quiet 22 8`) | - |

> **Note**: Hotkeys work in both encrypted and unencrypted sessions since they're handled client-side.
>
> **Notifications**: See [NOTIFICATIONS.md](NOTIFICATIONS.md) for full notification system documentation including desktop notifications, quiet hours, and focus mode.

### Plugin Commands (Admin Only)

Text commands and hotkeys for plugin management. See [Plugin Management hotkeys](#plugin-management-admin) for keyboard shortcuts.

| Command | Description | Hotkey |
|---------|-------------|--------|
| `:store` | Browse plugin store | `Alt+S` |
| `:plugin list` or `:list` | List installed plugins | `Alt+P` |
| `:plugin install <name>` or `:install <name>` | Install plugin | `Alt+I` |
| `:plugin uninstall <name>` or `:uninstall <name>` | Uninstall plugin | `Alt+U` |
| `:plugin enable <name>` or `:enable <name>` | Enable plugin | `Alt+E` |
| `:plugin disable <name>` or `:disable <name>` | Disable plugin | `Alt+D` |
| `:refresh` | Refresh plugin list from registry | `Alt+R` |

> **Note**: Both text commands and hotkeys work in E2E encrypted sessions (sent as admin messages that bypass encryption).

### File Sharing

**Direct send:**
```bash
:sendfile /path/to/file.txt
```

**Interactive picker:**
```bash
:sendfile
```
Navigate with arrow keys, Enter to select/open folders, ".. (Parent Directory)" to go up.

**Supported types:** Text, code, images, documents, archives (`.txt`, `.md`, `.json`, `.go`, `.py`, `.js`, `.png`, `.jpg`, `.pdf`, `.zip`, etc.)

## Keyboard Shortcuts

### General
| Key | Action |
|-----|--------|
| `Ctrl+H` | Toggle help overlay |
| `Enter` | Send message |
| `Alt+Enter` / `Ctrl+J` | Insert newline (multi-line input) |
| `Tab` | Autocomplete @mentions |
| `Esc` | Close menus / dialogs |
| `:q` | Quit application (vim-style) |
| `↑/↓` | Scroll chat |
| `PgUp/PgDn` | Page through chat |
| `Ctrl+C/V/X/A` | Copy/Paste/Cut/Select all |

### User Features
| Key | Action |
|-----|--------|
| `Alt+F` | Send file (file picker) |
| `Alt+C` | Create code snippet |
| `Ctrl+T` | Cycle themes |
| `Alt+T` | Toggle 12/24h time |
| `Alt+M` | Toggle message metadata (id / encrypted) on chat lines |
| `Alt+N` | Toggle desktop notifications |
| `Ctrl+L` | Clear chat history |

> **Multi-line input**: Use `Alt+Enter` or `Ctrl+J` to insert newlines. `Shift+Enter` is not reliably supported on Windows terminals.

### Admin Interface (Client)
| Key | Action |
|-----|--------|
| `Ctrl+U` | Select/cycle user |
| `Ctrl+D` | Database operations menu |
| `Ctrl+K` | Kick selected user |
| `Ctrl+B` | Ban selected user |
| `Ctrl+F` | Force disconnect selected user |
| `Ctrl+Shift+B` | Unban user (prompts for username) |
| `Ctrl+Shift+A` | Allow user (prompts for username) |

### Plugin Management (Admin)
| Key | Action |
|-----|--------|
| `Alt+P` | List installed plugins |
| `Alt+S` | View plugin store |
| `Alt+R` | Refresh plugin list |
| `Alt+I` | Install plugin (prompts for name) |
| `Alt+U` | Uninstall plugin (prompts for name) |
| `Alt+E` | Enable plugin (prompts for name) |
| `Alt+D` | Disable plugin (prompts for name) |

### Server
| Key | Action |
|-----|--------|
| `Ctrl+A` | Open terminal admin panel |

## Admin Panels

### Terminal Admin Panel
Enable with `--admin-panel` flag, then press `Ctrl+A` to access:
- Real-time server statistics (users, messages, performance)
- User management interface
- Plugin configuration
- Database operations
- Requires terminal environment (auto-disabled in systemd/non-terminal)

### Web Admin Panel
Enable with `--web-panel` flag, access at `http://localhost:8080/admin`:
- Secure session-based login (1-hour expiration)
- Live dashboard with metrics visualization
- RESTful API endpoints with `X-Admin-Key` auth
- CSRF protection on all state-changing operations
- HttpOnly cookies with SameSite protection

**API Example:**
  ```bash
curl -H "Cookie: admin_session=YOUR_SESSION" http://localhost:8080/admin/api/overview
  ```

## TLS Support

### When to Use TLS

- **Public deployments**: Server accessible from internet
- **Production environments**: Enhanced security required
- **Corporate networks**: Security policy compliance
- **HTTPS reverse proxies**: Behind nginx, traefik, **Caddy**, etc.

### Reverse proxy (Caddy)

The repo includes a **Docker Compose**-based Caddy setup for local or LAN use: **`docker-compose.proxy.yml`**, **`deploy/caddy/Caddyfile`**, **`deploy/caddy/proxy.env.example`** (and optional local **`deploy/caddy/proxy.env`**), and the walkthrough **[deploy/CADDY-REVERSE-PROXY.md](deploy/CADDY-REVERSE-PROXY.md)** (build flags, `config/.env`, firewall, `wss://` client flags, E2E, and **breaking change**: `config/.env` is applied with `godotenv.Overload` so file values override pre-set `MARCHAT_*` in the process environment).

**Quick reference:**

| Item | Role |
|------|------|
| `marchat-server` on the host | Listens on **8080** (`ws://`), reads **`config/.env`** |
| `docker compose -f docker-compose.proxy.yml up -d` | Runs Caddy; host **8443** → container **443** |
| Client | `wss://localhost:8443/ws` + `--skip-tls-verify` until you use a public CA cert on Caddy |
| If 8443 is refused | Caddy is not running; start the compose stack or use `ws://127.0.0.1:8080/ws` |

### Configuration Examples

**With TLS (production):**
```bash
# Generate self-signed cert (testing only)
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

export MARCHAT_ADMIN_KEY="your-key"
export MARCHAT_USERS="admin1,admin2"
export MARCHAT_TLS_CERT_FILE="./cert.pem"
export MARCHAT_TLS_KEY_FILE="./key.pem"
./marchat-server  # Shows wss:// in banner
```

**Without TLS (development):**
```bash
export MARCHAT_ADMIN_KEY="your-key"
export MARCHAT_USERS="admin1,admin2"
./marchat-server  # Shows ws:// in banner
```

**Client with TLS:**
```bash
# With verification (production)
./marchat-client --server wss://localhost:8080/ws

# Skip verification (dev/self-signed only)
./marchat-client --skip-tls-verify --server wss://localhost:8080/ws
```

> **Warning**: Use `--skip-tls-verify` only for development. Production should use valid CA-signed certificates.

## E2E Encryption

Global encryption for secure group chat using shared keys across all clients.

### How It Works
- **Shared Key Model**: All clients use same global encryption key for public channels
- **Simplified Management**: No complex per-user key exchange
- **X25519/ChaCha20-Poly1305**: Industry-standard encryption algorithms
- **Environment Variable**: `MARCHAT_GLOBAL_E2E_KEY` for key distribution
- **Auto-Generation**: Creates new key if none provided

### Setup Options

**Option 1: Shared Key (Recommended)**
```bash
# Generate 32-byte key
openssl rand -base64 32

# Set on all clients
export MARCHAT_GLOBAL_E2E_KEY="your-generated-key"

# Connect with E2E
./marchat-client --e2e --keystore-passphrase your-pass --username alice --server ws://localhost:8080/ws
```

**Option 2: Auto-Generate**
```bash
# Client generates and displays new key
./marchat-client --e2e --keystore-passphrase your-pass --username alice --server ws://localhost:8080/ws

# Output shows:
# [INFO] Generated new global E2E key (ID: RsLi9ON0...)
# [TIP] Set MARCHAT_GLOBAL_E2E_KEY=fF+HkmGArkPNsdb+... to share this key across clients
```

### Expected Output
```
[INFO] Using global E2E key from environment variable
E2E encryption enabled
Using global E2E key from environment variable
Global chat encryption: ENABLED (Key ID: RsLi9ON0...)
Encryption validation passed
E2E encryption enabled with keystore: config/keystore.dat
```

### Security Features
- **Forward Secrecy**: Unique session keys per conversation
- **Server Privacy**: Server cannot read encrypted messages
- **Local Keystore**: Encrypted with passphrase protection using PBKDF2
- **Validation**: Automatic encryption/decryption testing on startup

**Note**: Keystore encryption was upgraded from SHA256 to PBKDF2 for enhanced security. Existing keystores encrypted with the old method will need to be re-initialized.

## Plugin System

Extend functionality with remote plugins from configured registry.

### Configuration
```bash
# Default GitHub registry
export MARCHAT_PLUGIN_REGISTRY_URL="https://raw.githubusercontent.com/Cod-e-Codes/marchat-plugins/main/registry.json"

# Custom registry
export MARCHAT_PLUGIN_REGISTRY_URL="https://my-registry.com/plugins.json"
```

### Commands

**Text commands:**
```bash
:store                    # Browse available plugins
:plugin install echo      # Install plugin
:plugin list              # List installed
:plugin uninstall echo    # Remove plugin
:enable echo              # Enable installed plugin
:disable echo             # Disable plugin
:refresh                  # Refresh plugin registry
```

**Keyboard shortcuts** (Admin only):
- `Alt+P` - List installed plugins
- `Alt+S` - View plugin store  
- `Alt+R` - Refresh plugin list
- `Alt+I` - Install plugin (prompts for name)
- `Alt+U` - Uninstall plugin (prompts for name)
- `Alt+E` - Enable plugin (prompts for name)
- `Alt+D` - Disable plugin (prompts for name)

> **Note**: Plugin management commands and custom plugin commands (e.g., `:echo`) work in E2E encrypted sessions. See [Plugin Commands](#plugin-commands-admin-only) for full reference.

### Available Plugins
- **echo** (v2.0.1): Simple echo plugin for testing (provides `:echo` command)
- **weather** (v1.0.0): Get weather information and forecasts using wttr.in (`:weather [location]`, `:forecast [location]`)
- **githooks** (v1.0.0): Git repository management with status, log, branch, and diff commands (`:git-status`, `:git-log`, `:git-branch`, `:git-diff`, `:git-watch` admin-only)

See [PLUGIN_ECOSYSTEM.md](PLUGIN_ECOSYSTEM.md) for development guide.

## Moderation System

**Temporary Kicks (24 hours):**
- `:kick <username>` or `Ctrl+K` for temporary discipline
- Auto-allowed after 24 hours, or override early with `:allow`
- Ideal for cooling-off periods

**Permanent Bans (indefinite):**
- `:ban <username>` or `Ctrl+B` for serious violations
- Remains until manual `:unban` or `Ctrl+Shift+B`
- Ideal for persistent troublemakers

**Ban History Gaps:**
Prevents banned users from seeing messages sent during ban periods. Enable with `MARCHAT_BAN_HISTORY_GAPS=true` (disabled by default).

## Client Configuration

### Interactive Mode (Default)
```bash
./marchat-client
```
Guides through server URL, username, admin privileges, E2E encryption, theme selection, and profile saving.

### Quick Start Options
```bash
# Auto-connect to recent profile
./marchat-client --auto

# Select from saved profiles
./marchat-client --quick-start
```

### Profile Management
Profiles stored in platform-appropriate locations:
- **Windows**: `%APPDATA%\marchat\profiles.json`
- **macOS**: `~/Library/Application Support/marchat/profiles.json`  
- **Linux**: `~/.config/marchat/profiles.json`

**During profile selection:**
- `i` or `v` - View profile details
- `r` - Rename profile
- `d` - Delete profile

### Traditional Flags
```bash
# Basic connection
./marchat-client --server ws://localhost:8080/ws --username alice

# Admin connection
./marchat-client --server ws://localhost:8080/ws --username admin --admin --admin-key your-key

# E2E encrypted
./marchat-client --server ws://localhost:8080/ws --username alice --e2e --keystore-passphrase your-pass

# Non-interactive (requires all flags)
./marchat-client --non-interactive --server ws://localhost:8080/ws --username alice
```

## Security Best Practices

1. **Generate Secure Keys**
   ```bash
   # Admin key (64 hex characters)
   openssl rand -hex 32
   
   # Global E2E key (base64-encoded 32 bytes)
   openssl rand -base64 32
   ```

2. **Secure File Permissions**
   ```bash
   chmod 600 ./config/marchat.db    # Database
   chmod 600 ./config/keystore.dat  # Keystore
   chmod 700 ./config               # Config directory
   ```

3. **Production Deployment**
   - Use TLS (`wss://`) with valid CA-signed certificates
   - Deploy behind reverse proxy (nginx, traefik, or Caddy; see [deploy/CADDY-REVERSE-PROXY.md](deploy/CADDY-REVERSE-PROXY.md) for the bundled Caddy example)
   - Restrict server access to trusted networks
   - Use Docker secrets for sensitive environment variables
   - Enable rate limiting and brute force protection
   - Monitor security logs regularly

4. **E2E Encryption**
   - Store `MARCHAT_GLOBAL_E2E_KEY` securely
   - Use strong keystore passphrases
   - Never share keystores between users
   - Rotate keys periodically for sensitive deployments

5. **Username Allowlist (Optional)**
   ```bash
   # Restrict to specific users for private servers
   export MARCHAT_ALLOWED_USERS="alice,bob,charlie"
   ```
   - Usernames validated (letters, numbers, `_`, `-`, `.` only)
   - Max 32 characters, cannot start with `:` or `.`
   - Case-insensitive matching
   - Protects against log injection and command injection

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Wrong config folder / paths | Run `marchat-client -doctor` or `marchat-server -doctor`; see **Client vs server config locations** |
| Connection failed | Verify `ws://` or `wss://` protocol in URL |
| `wss://localhost:8443` reconnect loop / connection refused | Ensure Caddy is up: `docker compose -f docker-compose.proxy.yml up -d`, or use `ws://127.0.0.1:8080/ws` without the proxy ([reverse proxy guide](deploy/CADDY-REVERSE-PROXY.md)) |
| Admin commands not working | Check `--admin` flag and correct `--admin-key` |
| Clipboard issues (Linux) | Install xclip: `sudo apt install xclip` |
| Port in use | Change port: `export MARCHAT_PORT=8081` |
| Database migration fails | Check file permissions, backup before source build |
| PostgreSQL connection fails | Verify URL format: `postgres://user:pass@host:5432/db?sslmode=disable`; test with `psql` using same creds |
| MySQL connection fails | Verify DSN prefix `mysql:` and DSN body `user:pass@tcp(host:3306)/db?parseTime=true`; test with `mysql` CLI |
| SQL syntax error after backend switch | Ensure tables were created by the current server version and restart after changing `MARCHAT_DB_PATH` |
| Message history missing | Expected after updates - user states reset for ban/unban improvements |
| Ban history gaps not working | Ensure `MARCHAT_BAN_HISTORY_GAPS=true` (disabled by default) and `ban_history` table exists |
| TLS certificate errors | Use `--skip-tls-verify` for dev with self-signed certs |
| Plugin installation fails | Verify `MARCHAT_PLUGIN_REGISTRY_URL` is accessible and valid JSON |
| E2E encryption errors | Ensure `--e2e` flag and keystore passphrase provided, check debug logs |
| Global E2E key errors | Verify key is valid base64-encoded 32-byte key: `openssl rand -base64 32` |
| Blank encrypted messages | Fixed in v0.3.0-beta.5+ - ensure latest version |
| Username already taken | Use admin `:forcedisconnect <user>` or wait 5min for auto-cleanup |
| Stale connections | Server auto-cleans every 5min, or admin use `:cleanup` |
| Client frozen at startup | Fixed in latest - `--quick-start` uses proper UI |
| Multi-line input not working | Use `Alt+Enter` or `Ctrl+J`; `Shift+Enter` is not supported in most Windows terminals |

### Stale Connection Management

**Automatic:** Server detects and removes stale connections every 5 minutes using WebSocket ping.

**Manual (Admin):**
```bash
:cleanup                    # Clean all stale connections
:forcedisconnect username   # Force disconnect specific user
```

**Common scenarios:**
- Client crash/Ctrl+C: Auto-cleaned within 5 minutes
- Network interruption: Removed on next cleanup cycle
- Immediate reconnect: Admin uses `:forcedisconnect`

## Testing

Foundational test suite covering core functionality, cryptography, and plugins.

### Running Tests
```bash
go test ./...              # Run all tests
go test -cover ./...       # With coverage
go test ./server -v        # Specific package
go test ./... -timeout 10s # With timeout (CI recommended)
```

### Test Scripts
- **Linux/macOS**: `./test.sh`
- **Windows**: `.\test.ps1`

### Coverage Summary
| Package | Coverage | Size | Status |
|---------|----------|------|--------|
| `shared` | 85.9% | 348 LOC | High |
| `plugin/license` | 83.1% | 229 LOC | High |
| `client/crypto` | 79.5% | 354 LOC | High |
| `config` | 73.2% | 327 LOC | High |
| `client/config` | 54.5% | 1862 LOC | Medium |
| `plugin/store` | 47.0% | 552 LOC | Medium |
| `cmd/license` | 42.2% | 160 LOC | Medium |
| `server` | 33.7% | 6558 LOC | Medium |
| `plugin/manager` | 23.8% | 747 LOC | Low |
| `client` | 23.3% | 5334 LOC | Low |
| `plugin/host` | 21.1% | 617 LOC | Low |
| `cmd/server` | 5.3% | 455 LOC | Low |

**Overall: 34.1%** - See [TESTING.md](TESTING.md) for detailed information.

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- Development setup instructions
- Code style guidelines and conventions
- Pull request process and requirements
- Testing expectations

**Quick Start:**
```bash
git clone https://github.com/Cod-e-Codes/marchat.git
cd marchat
go mod tidy
go test ./...
```

## Documentation

- **[QUICKSTART.md](QUICKSTART.md)** - Short path from install to first client connection
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Components, data flow, config paths, diagnostics
- **[PROTOCOL.md](PROTOCOL.md)** - WebSocket message types and payloads
- **[deploy/CADDY-REVERSE-PROXY.md](deploy/CADDY-REVERSE-PROXY.md)** - Optional TLS reverse proxy (Caddy) for local or LAN `wss://`
- **[NOTIFICATIONS.md](NOTIFICATIONS.md)** - Notification system guide (desktop, quiet hours, focus mode)
- **[THEMES.md](THEMES.md)** - Custom theme creation guide
- **[PLUGIN_ECOSYSTEM.md](PLUGIN_ECOSYSTEM.md)** - Plugin development guide
- **[ROADMAP.md](ROADMAP.md)** - Planned features and enhancements
- **[TESTING.md](TESTING.md)** - Comprehensive testing guide
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Contribution guidelines
- **[SECURITY.md](SECURITY.md)** - Security policy and reporting
- **[CONTRIBUTORS.md](CONTRIBUTORS.md)** - Full contributor list

## Getting Help

- [Report bugs](https://github.com/Cod-e-Codes/marchat/issues)
- [Ask questions](https://github.com/Cod-e-Codes/marchat/discussions)
- Commercial support: [cod.e.codes.dev@gmail.com](mailto:cod.e.codes.dev@gmail.com)

## Appreciation

Thanks to [Self-Host Weekly](https://selfh.st/weekly/2025-07-25/), [mtkblogs.com](https://mtkblogs.com/2025/07/23/marchat-a-go-powered-terminal-chat-app-for-the-modern-user/), and [Terminal Trove](https://terminaltrove.com/) for featuring marchat!

See [CONTRIBUTORS.md](CONTRIBUTORS.md) for full contributor list.

---

**License**: [MIT License](LICENSE)
