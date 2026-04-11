# marchat

<img src="assets/marchat-transparent.svg" alt="marchat - terminal chat application" width="200" height="auto">

[![Go CI](https://github.com/Cod-e-Codes/marchat/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/Cod-e-Codes/marchat/actions/workflows/go.yml)
[![MIT License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Cod-e-Codes/marchat?logo=go)](https://go.dev/dl/)
[![GitHub all releases](https://img.shields.io/github/downloads/Cod-e-Codes/marchat/total?logo=github)](https://github.com/Cod-e-Codes/marchat/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/codecodesxyz/marchat?logo=docker)](https://hub.docker.com/r/codecodesxyz/marchat)
[![Version](https://img.shields.io/badge/version-v0.11.0--beta.5-blue)](https://github.com/Cod-e-Codes/marchat/releases/tag/v0.11.0-beta.5)

A lightweight terminal chat with real-time messaging over WebSockets, optional E2E encryption, and a flexible plugin ecosystem. Built for developers who prefer the command line.

**Quick start:** [QUICKSTART.md](QUICKSTART.md) for a single-page walkthrough (install → server → client → next docs).

## Latest Updates

### v0.11.0-beta.5 (Current)

**Released 2026-04-10.** Since **[v0.11.0-beta.4](https://github.com/Cod-e-Codes/marchat/releases/tag/v0.11.0-beta.4)**; compare [`v0.11.0-beta.4...v0.11.0-beta.5`](https://github.com/Cod-e-Codes/marchat/compare/v0.11.0-beta.4...v0.11.0-beta.5). Commits: **`git log v0.11.0-beta.4..v0.11.0-beta.5 --oneline`**.

- **Server**: RFC 6455 WebSocket close frames on handshake errors; hub stays off plugin IPC with bounded, best-effort, at-most-once plugin chat fan-out.
- **Client**: Experimental env-driven **exthook** and **`-doctor`** integration.
- **Plugin SDK**: **`RunStdio`** / **`HandlePluginRequest`** stdio loop; echo sample uses the SDK; docs and **README** plugin examples aligned (**GetConfig**, **Marshal**); **`plugin/sdk/cov`** gitignored; CI runs nested plugin modules (**fmt**, **govulncheck**).
- **Tests / CI**: Server loadverify benches and rate-limit coverage; **`-doctor`** tests use the injectable **`osEnviron`** hook under **`environMu`** (no parallel **`buildEnvLines`** tests that swap it); **plugin host** **`StopPlugin`** waits for stdout/stderr reader goroutines before reuse so **`-race`** is clean on disable/enable; Dependabot Node 20 note in **`.github/dependabot.yml`**.
- **Docs**: **TESTING** bench section; coverage/LoC tables refreshed from **`go test -coverprofile=mergedcoverage ./...`** (overall **38.1%** statements, main module); hook example lives under **`_example_hook`**; prose uses ASCII hyphens where edited.
- **Deps**: **`golang.org/x/crypto`**, **`golang.org/x/term`**, **`modernc.org/sqlite`**.

### v0.11.0-beta.4

**Released 2026-04-09.** [Compare from beta.3](https://github.com/Cod-e-Codes/marchat/compare/v0.11.0-beta.3...v0.11.0-beta.4). E2E edit consistency; deterministic theme cycle; security scanner vs **govulncheck** docs; **`.gitattributes`** LF normalization.

### v0.11.0-beta.3

**Released 2026-04-09.** [Compare from beta.2](https://github.com/Cod-e-Codes/marchat/compare/v0.11.0-beta.2...v0.11.0-beta.3). Keystore v3 and config/path fixes; web admin refresh; plugin SDK context and host fixes; DB smoke CI; Go 1.25.9; demos, E2E docs, and release asset workflow updates.

### v0.11.0-beta.2

Go 1.25.8 toolchain/docs; **`-doctor`** and env reflection improvements; terminal chrome and **`:msginfo`** metadata; license cache and server hardening; static release zips + **linux-arm64** for Termux.

### Earlier

- **v0.11.0-beta.1**: Multi-DB (SQLite / Postgres / MySQL), reactions, read receipts, message state, serialized WS writes, admin TUI ([PR #83](https://github.com/Cod-e-Codes/marchat/pull/83)).
- **v0.10.x**: Core chat features (edit/delete/pin/search, DMs, channels, E2E files, plugins), **`-doctor`**, Docker, Caddy TLS proxy docs ([**deploy/CADDY-REVERSE-PROXY.md**](deploy/CADDY-REVERSE-PROXY.md)), **`config/.env`** precedence.

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
- **E2E Encryption** - ChaCha20-Poly1305 with a shared global key (`MARCHAT_GLOBAL_E2E_KEY`), including file transfers
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
- **Secure**: Optional E2E encryption with ChaCha20-Poly1305 (global symmetric key)
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

Tables created by the server (dialect-aware DDL for SQLite, PostgreSQL, and MySQL):
- **messages**: Core message storage with `message_id`, encryption fields, edit/delete/pin flags
- **user_message_state**: Per-user message history state and last-seen timestamp
- **ban_history**: Ban/unban event tracking for history gaps
- **message_reactions**: Durable emoji reactions (unique per message + user + emoji)
- **user_channels**: Last channel per user, persisted across reconnects
- **read_receipts**: Per-user read receipt state tracking

## Installation

**Binary Installation:**
```bash
# Linux (amd64)
wget https://github.com/Cod-e-Codes/marchat/releases/download/v0.11.0-beta.5/marchat-v0.11.0-beta.5-linux-amd64.zip
unzip marchat-v0.11.0-beta.5-linux-amd64.zip && chmod +x marchat-*

# macOS (amd64)
wget https://github.com/Cod-e-Codes/marchat/releases/download/v0.11.0-beta.5/marchat-v0.11.0-beta.5-darwin-amd64.zip
unzip marchat-v0.11.0-beta.5-darwin-amd64.zip && chmod +x marchat-*

# Windows - PowerShell
iwr -useb https://raw.githubusercontent.com/Cod-e-Codes/marchat/main/install.ps1 | iex
```

**Package managers:**

```bash
# Homebrew (macOS / Linux): https://github.com/Cod-e-Codes/homebrew-marchat
brew tap cod-e-codes/marchat
brew install marchat
```

```powershell
# Scoop (Windows): https://github.com/Cod-e-Codes/scoop-marchat
scoop bucket add marchat https://github.com/Cod-e-Codes/scoop-marchat
scoop install marchat
```

**winget:** When [microsoft/winget-pkgs#358094](https://github.com/microsoft/winget-pkgs/pull/358094) is merged, install with `winget install Cod-e-Codes.Marchat`.

See [PACKAGING.md](PACKAGING.md) and `packaging/` for AUR, winget manifests, Chocolatey packaging templates (not published to the community gallery yet), and how releases line up with those channels.

**Docker:**
```bash
docker pull codecodesxyz/marchat:v0.11.0-beta.5
docker run -d -p 8080:8080 \
  -e MARCHAT_ADMIN_KEY=$(openssl rand -hex 32) \
  -e MARCHAT_USERS=admin1,admin2 \
  codecodesxyz/marchat:v0.11.0-beta.5
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
- Go 1.25.9 or later ([download](https://go.dev/dl/))
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
| `MARCHAT_GLOBAL_E2E_KEY` | No | - | Base64 32-byte global E2E key (server and/or client). On the **client**, if set, it **overrides** the key from `keystore.dat` for that run only; the keystore file is **not** updated. See [E2E Encryption](#e2e-encryption). |
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

**Keystore file:** The client uses `keystore.dat` under `MARCHAT_CONFIG_DIR` or the default app data directory when that file exists; if `MARCHAT_CONFIG_DIR` is set and has no keystore yet, it still uses an existing `keystore.dat` in the standard per-user marchat folder. Only after those checks does it use legacy `./keystore.dat` in the process working directory, so a stray file in a repo clone does not override your real profile keystore.

The repository’s `config/` directory holds **server** runtime files and the **Go package** `github.com/Cod-e-Codes/marchat/config`; it is not the client’s profile folder.

### Diagnostics (`-doctor`)

Run **`./marchat-client -doctor`** or **`./marchat-server -doctor`** for a text report (paths, redacted `MARCHAT_*` secrets as length-only, other env values as shown, sanity checks). **Server** doctor lists `MARCHAT_*` **after** loading the resolved config directory’s **`.env`** (same as the running server), so values are not limited to what your shell exported. **Client** doctor only shows variables present in the client process (it does not read the server’s `config/.env`); it also lists **experimental client hook** env vars and validates receive/send hook paths when set (see [CLIENT_HOOKS.md](CLIENT_HOOKS.md)). **Server** doctor does **not** list those client-only hook variables, even if they are set in the shell (for example when you run client and server from the same session). Server doctor also reports the detected DB dialect, validates the configured DB connection string format, and attempts a DB ping. On a **color-capable terminal** (stdout is a TTY), the text report uses **ANSI colors** aligned with the server pre-TUI banner; set **`NO_COLOR`** or redirect to a file/pipe for **plain** output. Use **`-doctor-json`** for machine-readable output (never colorized). If both flags were passed, `-doctor-json` wins. Exits without starting the TUI or listening on a port. See [ARCHITECTURE.md](ARCHITECTURE.md) for details.

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
| `:edit <id> <text>` | Edit your own message by ID (admins cannot edit others' messages; with E2E on, the new text is encrypted like normal chat and the server keeps `is_encrypted` in sync) |
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

### Plugin Commands

Plugin **management** (install, uninstall, enable, disable) is admin-only. Plugin **chat commands** (e.g. `:echo`, `:weather`) are available to all users unless the plugin manifest sets `AdminOnly: true`. See [Plugin Management hotkeys](#plugin-management-admin) for keyboard shortcuts.

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
- Sidebar-navigation layout (Overview, Users, System, Logs, Plugins, Metrics)
- Secure session-based login (1-hour expiration)
- Live dashboard with stats cards, configuration, and database info
- User management table with status badges, ban/kick with confirmation modals
- System management with config viewer and database statistics
- Real-time log viewer with level/component/timestamp columns
- Plugin management with install, enable, disable, uninstall actions
- Performance metrics with connection, message, and memory history
- RESTful API endpoints with session cookie auth
- CSRF protection on all state-changing operations (HttpOnly + SameSite cookies)
- Responsive design (sidebar collapses on mobile with hamburger menu)

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
- **Shared Key Model**: All clients use the same 32-byte global key for encrypted chat (and optional encrypted files)
- **Simplified Management**: No per-user public-key exchange on the wire; distribute the key out-of-band (env var or copy from first client)
- **ChaCha20-Poly1305**: Authenticated encryption for payloads; see **PROTOCOL.md** for the on-the-wire layout
- **Environment Variable**: `MARCHAT_GLOBAL_E2E_KEY` (base64) for key distribution
- **Auto-Generation**: First client can generate a key if none is provided (then share it to peers)

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
# Client generates a key and saves it to the encrypted keystore (raw key is not printed)
./marchat-client --e2e --keystore-passphrase your-pass --username alice --server ws://localhost:8080/ws

# Output shows Key ID only, e.g.:
# [INFO] Generated new global E2E key (ID: RsLi9ON0...)
# [TIP] The key is not printed ... (copy keystore.dat + passphrase, or pre-share MARCHAT_GLOBAL_E2E_KEY)
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
- **Server Privacy**: Server cannot read encrypted message bodies when E2E is used
- **Local Keystore**: Global key stored in a passphrase-protected file (PBKDF2 + AES-GCM); see **Keystore file format** below
- **No raw key on stdout**: Auto-generated global keys are not echoed in full (reduces exposure via logs, scrollback, or screen capture); only a Key ID is shown in `[INFO]`.
- **Validation**: Automatic encryption/decryption round-trip test on startup

### Keystore file format and `MARCHAT_GLOBAL_E2E_KEY`

- **On-disk format (current)**: `keystore.dat` is a small binary file: a fixed **magic** and **version**, a **random 16-byte salt** stored in the file, then **AES-GCM** ciphertext of the JSON payload (including the global ChaCha20-Poly1305 key). The passphrase is stretched with **PBKDF2** (SHA-256, 100k iterations) using that embedded salt. This means the same passphrase unlocks the file even if the absolute path to `keystore.dat` changes (for example after moving the file or when the client resolves a different config directory). Older files that derived PBKDF2 salt from the **keystore path** are still supported: on first successful unlock they are **rewritten** in the new format.
- **Environment variable vs file**: If **`MARCHAT_GLOBAL_E2E_KEY`** is set in the client process, that key is used for encryption/decryption for **this run**. The on-disk keystore is **not** modified. You will see **`[INFO] Using global E2E key from environment variable`**. If you later **unset** the variable, the client uses the key from `keystore.dat` again, so the effective key can appear to “change back” even though the file was never updated. To persist a shared key in the file, run **without** the env var once. When the client **auto-generates** a key, it **does not** print the raw base64 material (only a Key ID); share the key with other clients by copying **`keystore.dat`** and the **same passphrase**, or by agreeing on **`MARCHAT_GLOBAL_E2E_KEY`** beforehand (e.g. **`openssl rand -base64 32`** on a trusted machine).
- **Which file is used**: Same resolution order as in **Client vs server config locations** above (primary config dir → per-user marchat keystore when override is empty → cwd legacy).
- **Legacy note**: Keystore wrapping was previously upgraded to PBKDF2 (replacing an older derivation). Very old keystores from that era may still need re-initialization if they cannot be decrypted.

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

## Advanced features

### Experimental client hooks (local automation)

The TUI client can spawn **optional** external programs on send/receive and pass one JSON line on stdin per event, useful for custom logging, bridges, or local tooling. This is **experimental** (protocol may change); it does not replace server plugins or built-in notifications.

- **Documentation:** [CLIENT_HOOKS.md](CLIENT_HOOKS.md) (security model, env vars, JSON shape).
- **Entrypoint:** from the repo root, `go run ./client` (the client lives in `client/`, not the repo root).

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
| Wrong config folder / paths | Run `./marchat-client -doctor` or `./marchat-server -doctor` (add `-doctor-json` for scripts; set `NO_COLOR` for plain text). See **Client vs server config locations**. **Server:** if you set `MARCHAT_CONFIG_DIR` only in `config/.env`, restart after saving; the loader re-reads it after `Overload`. |
| Connection failed | Use `ws://` or `wss://` and the path your server uses (default HTTP handler is **`/ws`**, e.g. `ws://host:8080/ws`). |
| `wss://localhost:8443` reconnect loop / connection refused | Ensure Caddy (or your proxy) is up: `docker compose -f docker-compose.proxy.yml up -d`, or connect directly with `ws://127.0.0.1:8080/ws` ([reverse proxy guide](deploy/CADDY-REVERSE-PROXY.md)). |
| Admin commands not working | Client must use **`--admin`** and **`--admin-key`** matching the server’s `MARCHAT_ADMIN_KEY`; username must be listed in `MARCHAT_USERS`. |
| Clipboard issues (Linux) | Install a clipboard tool (e.g. `sudo apt install xclip` or `xsel`). |
| Port in use | Set `MARCHAT_PORT` (e.g. `8081`) in the environment or `config/.env` and restart the server. |
| Database migration fails | Check file permissions; back up the database before upgrades; run the **same** server binary version that created the schema. |
| PostgreSQL connection fails | Verify URL format: `postgres://user:pass@host:5432/db?sslmode=disable`; test with `psql` using the same credentials. |
| MySQL connection fails | Verify DSN prefix `mysql:` and body `user:pass@tcp(host:3306)/db?parseTime=true`; test with the `mysql` CLI. |
| SQL syntax error after backend switch | Ensure tables were created by the current server version and restart after changing `MARCHAT_DB_PATH`. |
| Message history looks incomplete | History depends on **channel**, **per-user message state**, and server filters. **Ban/unban** and related flows can reset stored state so scrollback differs from the raw DB. |
| Ban history gaps not working | Set `MARCHAT_BAN_HISTORY_GAPS=true` (default off). The server creates the **`ban_history`** table when using a database backend that runs marchat migrations. |
| TLS certificate errors | For dev/self-signed certs, pass **`--skip-tls-verify`** on the client (or enable **Skip TLS verify** in the profile / interactive setup). |
| Plugin installation fails | Check registry URL (`MARCHAT_PLUGIN_REGISTRY_URL`), network access, and JSON validity; commercial plugins need a valid license for the **plugin name** (see **PLUGIN_ECOSYSTEM.md**). |
| E2E encryption errors | Use **`--e2e`** and the keystore passphrase; see **[E2E Encryption](#e2e-encryption)** (keystore path, `MARCHAT_GLOBAL_E2E_KEY` vs file). Client and server must share the same global key material. |
| Global E2E key errors | Key must be **base64** encoding **32 raw bytes** (`openssl rand -base64 32`). **`MARCHAT_GLOBAL_E2E_KEY`** overrides the in-memory key for that process and is **not** written to the keystore file. |
| `:savefile` picks the wrong payload when names collide | Received files are stored per sender internally, but `:savefile <name>` matches **basename only**. If two users sent the same filename, which copy is saved is **not deterministic**; ask for distinct names or avoid duplicate basenames until disambiguation is exposed in the UI. |
| Send file / nothing happens | Check the footer (**Connected** vs **Disconnected**). If disconnected, `:sendfile` should report **Not connected to server**; reconnect, then retry (including **Alt+F** after a connection is up). |
| Username already taken | A live or **stale** session may still hold the name. Admin: **`:forcedisconnect <user>`**. Otherwise the server’s **~5 minute** WebSocket ping sweep removes broken clients (or run **`:cleanup`**). |
| Stale / ghost sessions | Same as above: wait for the ping sweep, run `:cleanup`, or `:forcedisconnect`. |
| Multi-line input not working | Use **Alt+Enter** or **Ctrl+J** in the input (plain **Enter** sends). **Shift+Enter** is unreliable on many Windows terminals. |
| Doctor / CI noise | For automated checks, use `-doctor-json`. Secret values are redacted to length only (no suffix). |

### Stale Connection Management

**Automatic:** The hub runs **`CleanupStaleConnections`** about every **5 minutes**: it sends a WebSocket **ping** per client; failures remove the client and free the username.

**Manual (admin, from the client):**

```text
:cleanup                    # Run stale check now for all clients
:forcedisconnect <user>     # Drop a specific connected user
```

**Typical cases:**
- Abrupt client exit: may linger until the next ping sweep or `:cleanup`.
- Half-open TCP: same; `:forcedisconnect` clears it immediately if the server still lists the user.
- Immediate reclaim of a name: use `:forcedisconnect` (do not rely on the sweep if you need instant reuse).

## Testing

Foundational test suite covering core functionality, cryptography, and plugins. CI (`.github/workflows/go.yml`) runs the full suite with the race detector and a separate **database-smoke** job against Postgres and MySQL (see [TESTING.md](TESTING.md)).

### Running Tests
```bash
go test ./...              # Run all tests (main module only)
go test -cover ./...       # With coverage
go test ./server -v        # Specific package
go test ./... -timeout 10s # With timeout (CI recommended)
cd plugin/sdk && go test ./...   # Nested SDK module (separate go.mod)
```

### Test Scripts
- **Linux/macOS**: `./test.sh`
- **Windows**: `.\test.ps1`

### Coverage Summary
Percentages are **statement coverage** from a merged profile (`go test -coverprofile=... ./...` then `go tool cover -func=...`). **Size** is non-test `.go` lines per package (approximate). See [TESTING.md](TESTING.md) for file-level tables and how to regenerate from your `coverage` / `coverage.out` file. The nested **`plugin/sdk`** module is measured separately (about **59%** statements); see [TESTING.md](TESTING.md) for the exact figure and commands.

| Package | Coverage | Size | Status |
|---------|----------|------|--------|
| `shared` | 88.1% | 253 LOC | High |
| `plugin/license` | 87.1% | 246 LOC | High |
| `client/crypto` | 80.3% | 387 LOC | High |
| `config` | 73.2% | 339 LOC | High |
| `plugin/host` | 64.6% | 721 LOC | Medium |
| `client/config` | 58.0% | 1993 LOC | Medium |
| `internal/doctor` | 52.5% | 809 LOC | Medium |
| `plugin/store` | 47.0% | 552 LOC | Medium |
| `cmd/license` | 42.2% | 160 LOC | Medium |
| `server` | 36.3% | 7217 LOC | Low |
| `plugin/manager` | 32.1% | 747 LOC | Low |
| `client/exthook` | 24.1% | 204 LOC | Low |
| `client` | 23.1% | 5555 LOC | Low |
| `cmd/server` | 13.7% | 484 LOC | Low |

**Overall: 38.1%** (main module packages only). See [TESTING.md](TESTING.md) for detailed information.

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
- **[PACKAGING.md](PACKAGING.md)** - Package manager installs (Homebrew, winget, Scoop, AUR), Chocolatey templates, and release alignment
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Components, data flow, config paths, diagnostics
- **[PROTOCOL.md](PROTOCOL.md)** - WebSocket message types and payloads
- **[deploy/CADDY-REVERSE-PROXY.md](deploy/CADDY-REVERSE-PROXY.md)** - Optional TLS reverse proxy (Caddy) for local or LAN `wss://`
- **[NOTIFICATIONS.md](NOTIFICATIONS.md)** - Notification system guide (desktop, quiet hours, focus mode)
- **[CLIENT_HOOKS.md](CLIENT_HOOKS.md)** - Experimental client-side external hooks (local automation)
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
