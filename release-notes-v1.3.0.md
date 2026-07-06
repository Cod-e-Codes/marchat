## v1.3.0

*Released: 6 July 2026*  
*Commit: 1b78dee*

### Client

- **Charm v2 migration**: Bubble Tea, Bubbles, and Lip Gloss on `charm.land/*/v2` (`tea.View`, `KeyPressMsg`, bubbles setters) (`client/main.go`, `client/render.go`)
- **Transcript wrap**: ANSI-aware word-wrap for chat bodies; long URLs wrap at path boundaries with hyperlink style and OSC 8 `Style.Hyperlink` hrefs (ASCII hyphens) (`client/render.go`)
- **Wrapped URL clicks**: Mouse click-to-open fallback for wrapped URLs; OSC 8 terminals should use embedded hyperlinks; copy from message otherwise ([#103](https://github.com/Cod-e-Codes/marchat/issues/103))
- **Reactions**: Aliases `thumbsup` / `thumbsdown` and commands `:unreact`, `:thumbsup`, `:thumbsdown` (`client/commands.go`)
- **E2E search**: When encryption is on and server search returns no matches, a `System` line notes ciphertext-only matching (`client/main.go`)
- **Composer and viewport**: **Fix** composer chrome, multiline keys, placeholder cursor, and scroll-to-tail follow; mouse wheel routes to the active viewport (`client/main.go`)
- **Overlays**: Help and DB overlays suppress typing, URL clicks, and read-receipt flush until closed (`client/main.go`)
- **System feedback**: Ephemeral `System` command feedback uses the banner; transcript notices stay channel-scoped (`client/main.go`, `client/render.go`)
- **Reconnect**: Backoff advances on failure; transcript notices, reactions, and read receipts stay channel-scoped after reconnect (`client/websocket.go`, `client/main.go`)
- **E2E logging**: E2E paths do not log plaintext (`client/`)

### Server

- **Handshake replay**: Replay up to 50 visible messages on every handshake, including reconnect (`server/hub.go`, `server/handlers.go`)
- **Channel stamping**: Outbound messages stamped to the sender's channel; typing, reactions, and read receipts channel-scoped (`server/hub.go`, `server/client.go`)
- **Postgres**: **Fix** boolean SQL for search, pin toggle, and pinned listing (`server/db_dialect.go`, `server/db.go`)
- **MySQL**: **Fix** `parseTime=true` when unset in DSN handling (`server/db.go`)
- **`:backup`**: SQLite-only; Postgres/MySQL use native backup tools (`server/`, docs)
- **Admin TUI**: Mouse scroll on tabs and tables (`server/admin_panel.go`)
- **`:cleardb`**: Clears `user_message_state` (`server/db.go`)

### Plugins

- **IPC**: **Fix** serialized plugin stdin writes so chat fan-out and command RPC cannot corrupt IPC lines (`plugin/host`, `plugin/manager`)

### Documentation

- **README**, **TESTING**, **ARCHITECTURE**, and **PROTOCOL** updated for channel stamping, reconnect, wrapped URL limitation ([#103](https://github.com/Cod-e-Codes/marchat/issues/103)), SQLite-only `:backup`, and plugin IPC
- **Tooling**: Project Agent skills under `.cursor/skills/` and `.cursor/rules/marchat.mdc`

### CI and packaging

- **Go**: **1.25.11** unchanged in **go.mod**, **go.yml**, **release.yml**, and **Dockerfile**
- **Dependencies**: **charm.land/bubbletea/v2** v2.0.8, **charm.land/bubbles/v2** v2.1.0, **charm.land/lipgloss/v2** v2.0.5 (replaces Charm v1); **github.com/charmbracelet/colorprofile** v0.4.3, **github.com/charmbracelet/x/ansi** v0.11.7, **github.com/lucasb-eyer/go-colorful** v1.4.0, **github.com/mattn/go-runewidth** v0.0.24; **github.com/jackc/pgx/v5** v5.10.0; **golang.org/x/crypto** v0.53.0; **golang.org/x/term** v0.44.0; **modernc.org/sqlite** v1.53.0

### Version and packaging

- **Install and build defaults**: **install.ps1**, **install.sh**, **build-release.ps1**, **`scripts/build-windows.ps1`**, and **`scripts/build-linux.sh`** download and build against **v1.3.0** release assets on GitHub.
- **Docs and metadata**: **SECURITY.md** and **README** (version badge, install snippets, Docker tag) reference **v1.3.0**; canonical Homebrew, Scoop, winget, Chocolatey, and AUR templates in **`packaging/`** target **v1.3.0** (refresh zip SHA256 from published assets before `choco pack` / manifest validation).
- **Post-release helper**: **`scripts/post-release-v1.3.0.ps1`** for Chocolatey checksum sync and manifest render after assets upload.

### Assets

- marchat-v1.3.0-linux-amd64.zip
- marchat-v1.3.0-linux-arm64.zip
- marchat-v1.3.0-windows-amd64.zip
- marchat-v1.3.0-darwin-amd64.zip
- marchat-v1.3.0-darwin-arm64.zip

**Full Changelog:** https://github.com/Cod-e-Codes/marchat/compare/v1.2.0...v1.3.0

### Breaking changes

- **Charm v2**: Client TUI depends on Charm **v2** modules (`charm.land/bubbletea/v2`, `bubbles/v2`, `lipgloss/v2`). Rebuild clients from this release; mixed old/new client builds against the same server are supported on the wire but TUI behavior may differ.
- **WebSocket JSON protocol**: No intentional breaking change; keystore and E2E wire encoding unchanged.
- **Wrapped URLs**: Click-to-open for long wrapped URLs is unreliable in some terminals; prefer OSC 8 hyperlinks or copy the URL ([#103](https://github.com/Cod-e-Codes/marchat/issues/103)).

### Migration guide

- **Binaries**: use **v1.3.0** archives from this release page, or **install.ps1** / **install.sh** with their default version.
- **Client / server**: restart after upgrade; no database schema changes in this release.
- **Postgres / MySQL operators**: benefit from boolean and `parseTime` fixes on upgrade; SQLite operators: note `:backup` remains SQLite-only.
- **Packaging maintainers**: run **`scripts/post-release-v1.3.0.ps1`** or **`packaging/ci/render-release-manifests.sh`** after zips publish to refresh SHA256 in **`packaging/`** and downstream manifests.
