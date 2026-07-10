# Marchat Test Suite

This document describes the test suite for the Marchat chat application.

## Overview

The Marchat test suite provides foundational coverage of the application's core functionality, including:

- **Unit Tests**: Testing individual components and functions in isolation
- **Integration Tests**: Testing the interaction between different components
- **Crypto Tests**: Testing cryptographic functions and E2E encryption
- **Database Tests**: Testing database operations and schema management
- **Server Tests**: Testing WebSocket handling, message routing, and user management

**Note**: Narrative release summaries live in **[CHANGELOG.md](CHANGELOG.md)** (and on GitHub Releases). This is a foundational test suite with good coverage for smaller utility packages and growing coverage for client and server components. **Overall statement coverage is 44.4%** across all packages in the main module, computed from the merged profile at the repo root (for example the `coverage` file or another path passed to `go test -coverprofile=... ./...`). Regenerate summaries with `go tool cover -func=<same-path>`. On **Windows PowerShell**, prefer a profile filename **without** a `.out` suffix (e.g. `mergedcoverage`, `coverage`, or per-package `ph` / `sv`) so the argument is not misparsed. The nested **`plugin/sdk`** module (separate `go.mod`) is not included in that merged number; its package statement coverage is **58.8%** when measured with `cd plugin/sdk && go test -coverprofile=sdkcover ./... && go tool cover -func=sdkcover`.

**Doctor env tests:** Production code reads process env via **`collectMarchatEnviron`** and the swappable **`osEnviron`** variable (see `internal/doctor/env.go`), guarded by **`environMu`**. Tests that replace **`osEnviron`** must capture the previous function while holding **`environMu`** and must not use **`t.Parallel()`** alongside other tests that also swap it, or **`go test -race`** can observe races and overlapping mocks. **`buildEnvLines`** tests are therefore sequential in the package.

**Database backends:** Automated tests open **SQLite** (usually in-memory or a temp file). PostgreSQL and MySQL/MariaDB are supported at runtime via `MARCHAT_DB_PATH`. **GitHub Actions** runs an extra **`database-smoke`** job (see `.github/workflows/go.yml`) with Postgres 16 and MySQL 8 service containers: it sets `MARCHAT_CI_POSTGRES_URL` and `MARCHAT_CI_MYSQL_URL` and runs `TestPostgresInitDBAndSchemaSmoke` / `TestMySQLInitDBAndSchemaSmoke` in `server/db_ci_smoke_test.go` (`InitDB` + `CreateSchema` + table checks + `GetRecentMessagesForUser` visible-replay SQL smoke + `SearchMessages` / pin toggle smoke). Locally, those tests **skip** unless you export the same variables (for MySQL, use a `mysql:` or `mysql://` prefix on the DSN so it is not parsed as a SQLite path). Schema creation is dialect-aware (including MySQL/MariaDB rules for indexed text).

**Release workflow (maintainers):** `.github/workflows/release.yml` uses a **`resolve-version`** job so the version string is available to Docker and all matrix legs (GitHub Actions does not support job outputs from matrix jobs). The **`build`** job sets **`CGO_ENABLED=0`** for static cross-compiled binaries. On a **published** release (not `workflow_dispatch`), **`upload-assets`** uploads all matrix **`.zip`** files with **`gh release upload`**, and the **`docker`** job appends Docker Hub pull instructions with **`gh release edit`** (GitHub CLI on the runner, **`GITHUB_TOKEN`**), avoiding third-party actions that still declare Node 20. **Termux** installs use the **linux-arm64** zip; `install.sh` / `install.ps1` map Android+aarch64 to that asset when needed.

## Manual testing gaps

Some client behavior is only verifiable in a real terminal emulator with mouse reporting enabled. CI and `go test` do not cover these paths today.

| Area | Automated coverage | Manual check |
|------|-------------------|--------------|
| Help panel scroll | `client/scroll_input_test.go` (`MouseWheelMsg` on help viewport) | Open help (Ctrl+H), wheel up/down and PgUp/PgDn on long content; confirm footer hint and scroll position change |
| DB admin menu scroll | `client/scroll_input_test.go` (target selection includes `showDBMenu`) | Admin: Ctrl+D, wheel or arrow keys on menu content; resize terminal and confirm menu still scrolls |
| Overlay input isolation | `client/scroll_input_test.go` (`overlayCapturesKeyboard`, read-receipt scoping) | With help open, type in chat area: no typing indicator; scroll help: no read receipt sent |
| Wrapped URL click-to-open | `client/render_test.go` asserts OSC 8 (`\x1b]8;;`) on wrapped segments; `client/main_test.go` covers fallback click helpers | In a narrow chat panel (Windows Terminal or iTerm), post a long GitHub commit URL, Ctrl+click or left-click the underlined link, confirm the browser opens the **full** URL (not a prefix such as `https://github.com/Cod`). See [#103](https://github.com/Cod-e-Codes/marchat/issues/103). |
| Lipgloss box + mouse coordinates | `chatPanelOrigin()` offset is unit-tested for banner/border rows only | Click URLs with and without banner text; confirm banner shows `[OK] Opening URL: ...` with the full href before the browser opens. |

**Workaround for users:** copy the URL from the message text when click-to-open opens a truncated link.

## Test Structure

### Test Files

| File | Description | Coverage |
|------|-------------|----------|
| `internal/doctor/doctor_test.go` | CLI diagnostics | Env masking, GitHub release JSON parsing, update check with fake HTTP transport, plain `-doctor` output when writer is not a terminal, server report env includes `.env` after load, ordered **`buildEnvLines`** / **`osEnviron`** hook tests without **`t.Parallel`** where the global mock is swapped |
| `internal/doctor/mask_test.go` | Env value masking | Full redaction of secret env values (no suffix leak), secret/non-secret key classification |
| `config/config_test.go` | Configuration loading and validation | Environment variables, validation rules, `MARCHAT_CONFIG_DIR` from `.env` |
| `shared/crypto_test.go` | Cryptographic operations | Key generation, encryption, decryption, session keys, nil-safety guards |
| `shared/types_test.go` | Data structures and serialization | Message types, JSON marshaling/unmarshaling |
| `client/crypto/keystore_test.go` | Client keystore management | Keystore initialization, encryption/decryption, file I/O |
| `client/config/config_test.go` | Client configuration management | Config loading/saving, path utilities, keystore migration |
| `client/config/interactive_ui_test.go` | Client interactive UI components | TUI forms, profile selection, authentication prompts |
| `client/code_snippet_test.go` | Client code snippet functionality | Text editing, selection, clipboard, syntax highlighting |
| `client/file_picker_test.go` | Client file picker functionality | File browsing, selection, size validation, directory navigation |
| `client/testmain_test.go` | Client test harness | Forces Lipgloss ANSI256 profile for headless hyperlink/render assertions |
| `client/render_test.go` | Message transcript rendering | URL wrap breakpoints, URL span markers, OSC 8 hyperlinks and underline on wrapped segments, continuation indent not underlined, system line severity, negative-ID transcript notice classification, headless URL click/index helpers (`buildTranscriptLineURLs`, `findURLAtTranscriptClick`; see [Manual testing gaps](#manual-testing-gaps)) |
| `client/scroll_input_test.go` | Scroll routing | Active viewport selection, help viewport wheel scroll, overlay input capture, read-receipt scoping to chat viewport, content height helper |
| `client/main_test.go` | Client main functionality | Message rendering, user lists, URL handling (single-line click miss vs hit, headless), encryption functions, flag validation, `wsConnected` transcript reset on reconnect, reconnect backoff doubling, `TestMessageIncrementsUnread`, client System prune/sort (`TestPruneEphemeralSystemMessages`, `TestPruneKeepsTranscriptSystemNotices`, `TestSortMessagesPersistedBeforeClientSystem`), reaction wire channel |
| `client/websocket_sanitize_test.go` | WebSocket URL / TLS hints | Sanitization helpers for display and connection hints |
| `client/websocket_e2e_test.go` | E2E DM and channel wire helpers | Encrypted outbound message shape, DM send wire format, decrypt roundtrip |
| `client/exthook/exthook_test.go` | Client hook helpers | Executable validation, hook JSON shaping, path rules |
| `internal/doctor/db_checks_test.go` | Doctor DB probes | SQLite connectivity and version checks used by `-doctor` |
| `cmd/server/main_test.go` | Server main function and startup | Flag parsing, configuration validation, TLS setup, admin normalization, `validateStartupConfig`, deprecated flags |
| `cmd/server/subprocess_doctor_test.go` | Server binary smoke | `go run ./cmd/server -doctor` / `-doctor-json` subprocess (covers `main` early exits) |
| `server/handlers_test.go` | Server-side request handling | Database operations, message insertion, visible handshake replay (`GetRecentMessagesForUser`), reconnect replay, DM limit under DM noise, IP extraction |
| `server/hub_test.go` | WebSocket hub management | User bans, kicks, connection management, non-blocking send verification |
| `server/loadverify_ratelimit_test.go` | WebSocket read-pump rate limit | Window, burst (20), and cooldown behavior (same constants as `client.go`) |
| `server/loadverify_bench_test.go` | Hub broadcast benchmarks (optional) | Channel vs system-wide fan-out, parallel senders, JSON marshal baseline; see [Optional hub load benchmarks](#optional-hub-load-benchmarks-server) |
| `server/integration_test.go` | End-to-end workflows | Message flow, ban flow, WebSocket handshake replay on reconnect (`TestIntegrationWebSocketHandshakeReplayOnReconnect`), concurrent operations |
| `server/admin_web_test.go` | Admin web interface | HTTP endpoints, authentication, admin panel functionality |
| `server/config_ui_test.go` | Server configuration UI | Configuration management, environment handling |
| `server/admin_panel_test.go` | Admin panel functionality | Admin-specific operations and controls |
| `server/db_test.go` | Database operations | Database initialization, schema setup |
| `server/db_dialect_test.go` | SQL dialect helpers | DSN detection, Postgres placeholder rebinding, boolean SQL literals, MySQL DSN via `mysql.Config` |
| `server/db_ci_smoke_test.go` | CI DB smoke | Postgres/MySQL `InitDB`, `CreateSchema`, core tables, visible handshake replay query, search and pin SQL (env-gated) |
| `server/message_state_test.go` | Durable reactions | Reaction persistence and replay helpers |
| `server/config_test.go` | Server configuration | Server configuration logic and validation |
| `server/client_test.go` | Server client management | WebSocket client initialization, message handling, admin operations, unknown admin command system reply (`TestHandleCommandUnknownAdminSendsSystemReply`), channel stamping (`TestStampClientChannelOverwritesSpoofedChannel`), non-admin unknown vs admin-only command replies |
| `server/health_test.go` | Server health monitoring | Health checks, system metrics, HTTP endpoints, concurrent access |
| `plugin/sdk/plugin_test.go` | Plugin SDK | Message types, extended fields (channel, encrypted, message_id, recipient, edited), JSON serialization, omitempty validation, backwards-compat unknown-field handling |
| `plugin/sdk/stdio_test.go` | Plugin SDK stdio | `HandlePluginRequest` / `RunIO` (init, message, command, shutdown), EOF handling |
| `plugin/host/host_test.go` | Plugin Host | Plugin lifecycle, communication, enable/disable |
| `plugin/host/outbound_test.go` | Plugin Host | Chat fan-out queue: drop when full, drain-on-stop, nil-queue safety |
| `plugin/host/plugin_lifecycle_test.go` | Plugin Host subprocess IPC | Minimal JSON plugin built with `go build`, `StartPlugin` / `StopPlugin`, `ExecuteCommand`, double-start guard |
| `plugin/store/store_test.go` | Plugin Store | Registry management, platform resolution, filtering |
| `plugin/fileurl/fileurl_test.go` | Plugin file URLs | Linux/Windows `file://` parsing, localhost host, unsupported hosts |
| `plugin/manager/manager_test.go` | Plugin Manager | `validatePluginName`, plugin state fallbacks, download checksum and traversal guards, archive binary execute bit, failed install leaves no empty dir, staged install rollback, installation, uninstallation, command execution |
| `plugin/manager/manager_lifecycle_test.go` | Plugin Manager + host | Discovery auto-start, `DisablePlugin` / `EnablePlugin`, `ExecuteCommand` delegation |
| `plugin/integration_test.go` | Plugin Integration | End-to-end plugin system workflows |
| `shared/version_test.go` | Version information | Version functions, variable validation, format consistency |
| `plugin/license/validator_test.go` | License validation | Signature verification, caching, expiration, tampered cache, plugin/cache key mismatch, file-path plugin name binding, error propagation from `IsLicenseValid` |
| `cmd/license/main_test.go` | License CLI tool functions | CLI functions (validateLicense, generateLicense, generateKeyPair, checkLicense) |

Per-file statement percentages for important paths are listed under [Test Coverage Areas](#test-coverage-areas) (from the same merged profile).

### Test Categories

#### 1. Unit Tests
- **Crypto Functions**: Key generation, encryption/decryption, session key derivation
- **Data Types**: Message structures, JSON serialization, validation
- **Utility Functions**: IP extraction, message sorting, database stats
- **Configuration**: Environment variable parsing, validation rules
- **Client Keystore**: Keystore initialization, encryption/decryption, file operations, passphrase handling
- **Client Config**: Configuration loading/saving, path utilities, keystore migration
- **Client Interactive UI**: TUI forms, profile selection, authentication prompts, navigation, validation
- **Client Code Snippet**: Text editing, selection, clipboard operations, syntax highlighting, state management
- **Client Main**: Message rendering, user lists, URL handling, encryption functions, flag validation, reconnect transcript handling (`TestWsConnectedClearsTranscript`)
- **Client WebSocket helpers**: URL / TLS hint sanitization (`websocket_sanitize_test.go`); E2E text/DM wire encode and decode (`websocket_e2e_test.go`)
- **Client Hooks (`client/exthook`)**: Executable path validation and hook-safe JSON for send/receive events (`exthook_test.go`)
- **Client File Picker**: File browsing, directory navigation, file selection, size validation, error handling
- **Doctor (`internal/doctor`)**: Server/client diagnostics, env masking, update checks, DB connectivity probes
- **Server Main**: Flag parsing, multi-flag handling, banner display, startup config validation (`validateStartupConfig`), admin username normalization (`normalizeAndValidateAdmins`), subprocess `-doctor` / `-doctor-json` smoke (`subprocess_doctor_test.go`)
- **Plugin Host (lifecycle)**: Minimal JSON plugin built with `go build`, `StartPlugin` / `StopPlugin`, `ExecuteCommand` (`plugin_lifecycle_test.go`)
- **Server Admin Web**: HTTP endpoints, authentication, admin panel functionality, web interface
- **Server Configuration**: Configuration management, environment handling, UI components
- **Server Database**: Database initialization, schema setup, connection management
- **Server Admin Panel**: Admin-specific operations, user management, administrative controls

#### 2. Integration Tests
- **Message Flow**: Complete message lifecycle from insertion to retrieval
- **User Management**: Ban/kick/unban workflows with database persistence
- **Handshake replay**: WebSocket connect/disconnect/reconnect receives persisted history on the wire (`TestIntegrationWebSocketHandshakeReplayOnReconnect`)
- **Concurrent Operations**: Thread-safe operations and race condition testing
- **Database Operations**: Schema creation, message capping, backup functionality

#### 3. Server Tests
- **WebSocket Handling**: Connection management, authentication, message routing
- **Hub Management**: User registration, broadcasting, cleanup operations
- **Admin Functions**: Ban management, user administration, system commands
- **Server Main Function**: Flag parsing, configuration validation, startup requirements (admins, key, port), TLS setup, admin username normalization, banner display
- **Client Management**: WebSocket client initialization, message handling, admin operations, connection settings
- **Health Monitoring**: Database health checks, system metrics collection, HTTP endpoints, concurrent access safety

## Running Tests

### Prerequisites

- Go 1.25.12 or later
- SQLite support (built into Go)
- PowerShell (for Windows test script)

### Basic Test Execution

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./server
go test ./shared
# Nested modules (separate go.mod; not run by root `go test ./...`)
cd plugin/sdk
go test ./...
cd ../examples/echo
go test ./...   # optional sample plugin (may report [no test files])
```

**Client hook sample binary:** `client/exthook/_example_hook` is a small **`package main`** logger used with [CLIENT_HOOKS.md](CLIENT_HOOKS.md). The leading **`_`** on the directory name follows the Go **`./...`** omission rules so root **`go test ./...`** does not compile it as a second package under `client/exthook` (build it only with an explicit path, for example **`go build ./client/exthook/_example_hook`**).

### Optional hub load benchmarks (server)

`server/loadverify_bench_test.go` defines `BenchmarkLoadverify_*` helpers for profiling hub broadcast paths. **`go test ./...` does not run benchmarks** unless you pass `-bench` (and usually `-run=^$` so only benchmarks execute).

What they approximate:

- **Hub `Run` loop** fed by `hub.broadcast`, with clients registered and channel membership like production, but **no real WebSocket** and **large per-client send buffers** so the harness measures routing/coordination rather than production backpressure.
- **`TypingMessage`** on the broadcast path avoids `SendMessageToPlugins` (see file comments). A separate sub-benchmark times `json.Marshal` on a text-shaped message for comparison.

Interpreting results: channel-scoped delivery still iterates **all** registered clients in `hub.go` and filters by channel; the `fixedChannel8` sub-benchmarks vary total clients while keeping eight members in `#bench` to highlight that cost scales with **server-wide** connections, not only room size. ns/op and B/op depend on hardware, OS, and Go version, so use these runs for trends and profiling, not as fixed targets.

Examples (repo root; adjust `-bench` regex as needed):

```bash
go test ./server -run=Loadverify -v
go test ./server -run='^$' -bench=Loadverify -benchmem -count=5
```

**Windows PowerShell:** quote `-cpuprofile` (e.g. `-cpuprofile="loadverify-cpu.pprof"`) so the path is not misparsed; the profile is written to the shell’s current directory unless you pass an absolute path.

```powershell
go test ./server -run='^$' -bench=Loadverify_HubBroadcast_ChannelMessage/all_in_channel_128 -cpuprofile="loadverify-cpu.pprof"
go tool pprof -top .\loadverify-cpu.pprof
```

### Using Test Scripts

#### Linux/macOS
```bash
# Run the test suite
./test.sh
```

#### Windows PowerShell
```powershell
# Run basic test suite
.\test.ps1

# Run with coverage report
.\test.ps1 -Coverage

# Run with verbose output
.\test.ps1 -Verbose
```

### Test Coverage

Generate and view test coverage:

```bash
# Generate merged profile (use a name without ".out" on Windows PowerShell)
go test -coverprofile=mergedcoverage ./...

# View coverage in terminal (same path as -coverprofile)
go tool cover -func=mergedcoverage

# Generate HTML coverage report
go tool cover -html=mergedcoverage -o coverage.html

# View coverage percentages directly (per package)
go test -cover ./...
```

Per-file statement percentages in the **Detailed File Coverage** table use the same `mergedcoverage` profile: each block contributes its statement count when the block execution count is greater than zero (same rules as `go tool cover`). Regenerate after `go test -coverprofile=mergedcoverage ./...`:

```powershell
# PowerShell (repo root)
$mod = "github.com/Cod-e-Codes/marchat/"
Get-Content mergedcoverage | ForEach-Object {
  if ($_ -match '^(.+\.go):') {
    $full = $Matches[1]
    if (-not $full.StartsWith($mod)) { return }
    $rel = $full.Substring($mod.Length)
    $p = $_ -split '\s+'
    $n = [int]$p[-2]; $cnt = [int]$p[-1]
    if (-not $files.ContainsKey($rel)) { $files[$rel] = @{s=0; c=0} }
    $files[$rel].s += $n
    if ($cnt -gt 0) { $files[$rel].c += $n }
  }
} -begin { $files = @{} } -end {
  $files.GetEnumerator() | Sort-Object Name | ForEach-Object {
    $pct = [math]::Round(100.0 * $_.Value.c / $_.Value.s, 1)
    "{0} {1}%" -f $_.Key, $pct
  }
}
```

Nested **`plugin/sdk`**: `cd plugin/sdk && go test -coverprofile=sdkcover ./... && go tool cover -func=sdkcover` (overall on the last line).

## Test Coverage Areas

### Current Coverage Status

| Package | Coverage | Status |
|---------|----------|--------|
| `shared` | 88.1% | High |
| `plugin/license` | 87.1% | High |
| `client/crypto` | 80.3% | High |
| `config` | 73.2% | High |
| `plugin/host` | 64.6% | Medium |
| `client/config` | 58.0% | Medium |
| `internal/doctor` | 65.9% | Medium |
| `plugin/store` | 47.1% | Medium |
| `cmd/license` | 42.2% | Medium |
| `server` | 43.4% | Low |
| `plugin/manager` | 66.1% | Medium |
| `plugin/fileurl` | 80.8% | High |
| `client/exthook` | 24.1% | Low |
| `client` | 29.4% | Low |
| `cmd/server` | 13.7% | Low |

**Overall coverage: 44.0%** (all packages in the main module; merged profile `mergedcoverage` from `go test -coverprofile=mergedcoverage ./...`, total from `go tool cover -func=mergedcoverage`)

### High Coverage (70%+)
- **Shared Package**: Cryptographic operations, data types, message handling, version utilities (88.1%)
- **Plugin License Package**: License validation, signature verification, caching, plugin name binding (87.1%)
- **Client Crypto Package**: Keystore management, encryption/decryption, file operations, raw encrypt/decrypt for file transfers (80.3%)
- **Config Package**: Configuration loading, validation, environment variables (73.2%)

### Medium Coverage (40-70%)
- **Plugin Host Package**: Load/start/stop lifecycle, JSON IPC with a minimal test plugin, `ExecuteCommand`, bounded async chat fan-out to plugins, serialized stdin writes (`stdinMu`), **stop** waits for IPC reader goroutines before clearing instance state (66.0%)
- **Client Config Package**: Configuration management, path utilities, keystore migration, interactive UI (58.0%)
- **Doctor Package**: Server/client diagnostics, env checks, update metadata, DB probes (66.5%)
- **Plugin Store**: Registry management, platform resolution, filtering, caching (47.0%)
- **Command License**: CLI functions for license management (42.2%)

### Low Coverage (<40%)
- **Server Package**: WebSocket handling, origin/IP request context (`request_context.go`), admin panel, database operations, message edit/delete/pin/search, channels, DMs (43.5%)
- **Plugin Manager**: Installation, store download paths, checksum and archive guards, enable/disable with persisted state (66.1%)
- **Client Hooks (`client/exthook`)**: Hook path validation and hook-safe message JSON (24.1%)
- **Client Package**: Message rendering, user lists, encryption functions (including E2E DM wire helpers), flag validation, TUI entrypoints (31.3%)
- **Server Main (`cmd/server`)**: Full `main` startup, HTTP/TLS serving, admin panel wiring (13.7% for `cmd/server`); helpers such as `normalizeAndValidateAdmins` and `validateStartupConfig` plus `-doctor` / `-doctor-json` subprocess smoke tests cover parts of `main`

### Detailed File Coverage
Statement percentages below use the same `mergedcoverage` profile as `go tool cover -func=mergedcoverage` (see **Test Coverage** for per-file regeneration). Listed files are either high-signal (≥40% and larger sources) or important entrypoints even when lower.

| File | Coverage | Package | Description |
|------|----------|---------|-------------|
| `shared/version.go` | 100.0% | shared | Version information functions |
| `internal/doctor/db_checks.go` | 100.0% | internal/doctor | SQLite checks for `-doctor` |
| `internal/doctor/env.go` | 97.4% | internal/doctor | Ordered `MARCHAT_*` env lines for doctor reports (`osEnviron` injectable under `environMu`) |
| `client/file_picker.go` | 98.3% | client | File selection TUI component |
| `server/health.go` | 89.3% | server | Health monitoring and status |
| `plugin/license/validator.go` | 87.1% | plugin/license | License validation and verification |
| `client/render.go` | 84.3% | client | TUI rendering helpers |
| `client/crypto/keystore.go` | 80.3% | client/crypto | Keystore management, raw encrypt/decrypt |
| `shared/crypto.go` | 82.5% | shared | Cryptographic operations |
| `server/db.go` | 70.2% | server | Database operations |
| `config/config.go` | 73.2% | config | Configuration management |
| `client/notification_manager.go` | 67.5% | client | Desktop / notification integration |
| `client/config/interactive_ui.go` | 66.9% | client/config | Interactive configuration UI |
| `plugin/host/host.go` | 66.0% | plugin/host | Plugin subprocess lifecycle, serialized stdin IPC |
| `server/logger.go` | 61.4% | server | Logging functionality |
| `server/message_state.go` | 59.2% | server | Reactions, read receipts, channel prefs |
| `server/db_dialect.go` | 74.1% | server | SQL dialect helpers |
| `server/hub.go` | 73.7% | server | WebSocket hub management, channels, DMs |
| `client/code_snippet.go` | 52.9% | client | Code snippet TUI component |
| `server/request_context.go` | 77.3% | server | WebSocket Origin checks, trusted-proxy client IP |
| `server/handlers.go` | 60.9% | server | HTTP/WebSocket handlers, edit/delete/pin/search |
| `plugin/store/store.go` | 47.1% | plugin/store | Plugin store operations |
| `cmd/license/main.go` | 42.2% | cmd/license | License CLI tool |
| `client/config/config.go` | 43.2% | client/config | Client configuration |
| `internal/doctor/doctor.go` | 57.1% | internal/doctor | Doctor orchestration and reporting |
| `server/config_ui.go` | 36.0% | server | Server configuration UI |
| `server/admin_web.go` | 33.3% | server | Admin web interface |
| `plugin/manager/manager.go` | 66.1% | plugin/manager | Plugin management |
| `server/admin_panel.go` | 20.0% | server | Admin panel functionality |
| `server/client.go` | 26.3% | server | Client management, message type routing |
| `cmd/server/main.go` | 13.7% | cmd/server | Server main application |
| `server/plugin_commands.go` | 12.7% | server | Plugin command handling |
| `client/exthook/exthook.go` | 24.1% | client/exthook | Client hook path checks and hook JSON |
| `client/main.go` | 12.8% | client | Client main application |

### Areas for Future Testing
- **Server Package**: Advanced WebSocket handling, complex message routing scenarios (current: 43.4%)
- **Client Package**: WebSocket communication, full TUI integration (current: 29.4%)
- **Plugin Host**: Broader command/response paths and failure modes beyond the minimal IPC test plugin (current: 64.6%)
- **Plugin Manager**: Store download, checksum, archive extraction, and install edge cases (current: 66.1%)
- **Server Main**: Full `main` execution, HTTP/TLS serving, admin panel integration (current: 13.7% statement coverage for `cmd/server/main.go`)
- **File Transfer**: File upload/download functionality
- **Client Hooks**: Broader hook execution paths and failure modes (current: 24.1% for `client/exthook`)
- **Plugin License**: License validation and enforcement (package: 87.1%; `validator.go` file line in the table above)

## Test Data and Fixtures

### Database Tests
- Uses in-memory SQLite databases for isolation
- Creates fresh schema for each test
- Tests both encrypted and plaintext messages
- Verifies message ordering and retrieval
- **Postgres / MySQL (CI)**: `server/db_ci_smoke_test.go` runs `InitDB` + `CreateSchema` against real servers when `MARCHAT_CI_POSTGRES_URL` / `MARCHAT_CI_MYSQL_URL` are set (see `.github/workflows/go.yml`). Smoke covers visible handshake replay, `SearchMessages`, and pin toggle/list SQL.

### Cryptographic Tests
- Tests ChaCha20-Poly1305 encrypt/decrypt for text payloads (`shared/crypto_test.go`)
- **Client Keystore**: Tests keystore initialization, global key load, encryption/decryption, file operations, passphrase handling, legacy path-salted file migration to the embedded-salt format, and loading the same file after changing its path

### Message Tests
- Tests various message types (text, file, admin)
- Verifies JSON serialization/deserialization
- Tests message ordering and timestamp handling
- Validates encrypted message handling

### Server Main Tests
- Tests command-line flag parsing and validation
- Verifies multi-flag functionality for admin users
- Tests configuration loading and environment variable handling
- Validates TLS configuration and WebSocket scheme selection
- Tests admin username normalization and duplicate detection (including empty or whitespace-only entries)
- Tests startup validation helpers (`validateStartupConfig`) for admins, admin key, and listen port
- Verifies banner display functionality
- Tests deprecated flag warnings and backward compatibility

### Plugin Manager Tests
- Tests `validatePluginName` for allowed names and rejection of traversal, invalid characters, length, and casing
- Tests `loadPluginState` when the state file is missing, corrupted, or valid JSON with a nil `enabled` map
- Tests discovery auto-start, `DisablePlugin` / `EnablePlugin`, and `ExecuteCommand` with a minimal built plugin (`manager_lifecycle_test.go`)

## Continuous Integration

The test suite is designed to run in CI/CD environments:

- **Default job** (`.github/workflows/go.yml` `build`): `gofmt -l` must be empty on the root tree, then `go test -race ./...`, `go vet ./...`, **`govulncheck -show verbose ./...`**, and **`golangci-lint run ./...`** (SQLite only for DB tests; DB smoke skips without env). Then **`plugin/sdk`** and **`plugin/examples/echo`** each run `go mod tidy`, `gofmt -l`, `go build ./...`, `go test -race ./...`, `go vet ./...`, **`govulncheck`**, and **`golangci-lint`** (nested modules use their own `go.mod` and are not included in root `./...`).
- **Database smoke job** (`database-smoke`): Postgres 16 and MySQL 8 services, then `go test -race ./server -run 'Test(Postgres|MySQL)InitDBAndSchemaSmoke'` with `MARCHAT_CI_POSTGRES_URL` / `MARCHAT_CI_MYSQL_URL` set.
- **Parallel Safe**: Standard tests avoid shared mutable global state; subprocess tests serialize via their own `go run` invocations. **Doctor** tests that swap **`osEnviron`** do not use **`t.Parallel()`** with each other. **Plugin host** **`StopPlugin`** joins stdout/stderr readers before clearing instance fields so lifecycle tests stay race-clean.
- **Deterministic**: Doctor subprocess tests set `MARCHAT_DOCTOR_NO_NETWORK=1` to avoid GitHub API flakiness.

## Adding New Tests

### Guidelines

1. **Test Naming**: Use descriptive test names that explain the scenario
2. **Test Structure**: Follow the Arrange-Act-Assert pattern
3. **Isolation**: Each test should be independent and not rely on other tests
4. **Coverage**: Aim for meaningful coverage, not just line coverage
5. **Documentation**: Add comments for complex test scenarios

### Example Test Structure

```go
func TestFeatureName(t *testing.T) {
    // Arrange
    setup := createTestSetup()
    input := createTestInput()
    
    // Act
    result, err := functionUnderTest(input)
    
    // Assert
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    
    if result != expectedResult {
        t.Errorf("Expected %v, got %v", expectedResult, result)
    }
}
```

### Database Test Pattern

```go
func TestDatabaseOperation(t *testing.T) {
    // Create test database
    db, err := sql.Open("sqlite", ":memory:")
    if err != nil {
        t.Fatalf("Failed to open test database: %v", err)
    }
    defer db.Close()
    
    // Create schema
    CreateSchema(db)
    
    // Test the operation
    // ... test implementation
}
```

## Performance Considerations

- **In-Memory Databases**: Tests use SQLite in-memory mode for speed
- **Parallel Execution**: Tests are designed to run in parallel when possible
- **Minimal Setup**: Each test creates only the data it needs
- **Fast Cleanup**: Tests clean up resources immediately

## Troubleshooting

### Common Issues

1. **Import Errors**: Ensure all dependencies are properly imported
2. **Database Locks**: Tests use separate in-memory databases to avoid conflicts
3. **Race Conditions**: All concurrent tests use proper synchronization
4. **Memory Leaks**: Tests properly close database connections and channels
5. **PowerShell Execution Policy**: May need to enable script execution: `Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser`
6. **Coverage Tool Syntax**: Use `go tool cover -func=coverage.out` or `-func=coverage` (path must match your profile file)

### Debug Mode

Run tests with debug output:

```bash
# Enable Go test debugging
go test -v -race ./...
```

### Local validation (PowerShell)

From the repo root, with `golangci-lint` and `govulncheck` on `PATH` (install with `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` and `go install golang.org/x/vuln/cmd/govulncheck@latest`):

```powershell
$env:GOTOOLCHAIN = "auto"
gofmt -w .
golangci-lint run ./...
go test ./... -count=1
go vet ./...
govulncheck -show verbose ./...
foreach ($D in @("plugin\sdk", "plugin\examples\echo")) {
  Push-Location $D
  gofmt -w .
  golangci-lint run ./...
  go test ./... -count=1
  go vet ./...
  govulncheck -show verbose ./...
  Pop-Location
}
```

## Contributing

When adding new functionality to Marchat:

1. **Write Tests First**: Follow TDD principles where possible
2. **Update This Document**: Add new test categories and coverage areas
3. **Maintain Coverage**: Ensure new code is properly tested
4. **Run Full Suite**: Always run all tests before submitting changes

## Test Metrics

- **Top-level tests**: 373 `Test*` entrypoints from `go test -list . ./...` on the main module; the nested **`plugin/sdk`** module adds 22 more (`cd plugin/sdk && go test -list . ./...`).
- **Test files**: 46 tracked `_test.go` files (`git ls-files '*_test.go'`), including `plugin/sdk/plugin_test.go` and `plugin/sdk/stdio_test.go` in the nested SDK module.
- **Packages (`go list ./...`)**: 15 in the main module; `plugin/sdk` and `plugin/examples/echo` are nested modules with their own `go.mod` files (root `go test ./...` does not run their tests).
- **Coverage by Package** (statement %, `go test -cover ./...`): 88.1% (`shared`), 87.1% (`plugin/license`), 80.8% (`plugin/fileurl`), 80.3% (`client/crypto`), 73.2% (`config`), 66.1% (`plugin/manager`), 64.6% (`plugin/host`), 58.0% (`client/config`), 65.9% (`internal/doctor`), 47.1% (`plugin/store`), 42.2% (`cmd/license`), 43.4% (`server`), 29.4% (`client`), 24.1% (`client/exthook`), 13.7% (`cmd/server`)
- **Nested `plugin/sdk` coverage**: **58.8%** statements (run inside `plugin/sdk`; not part of the root merged profile).
- **Overall Coverage**: **44.0%** across main-module packages (regenerate with `go test -coverprofile=mergedcoverage ./...` then `go tool cover -func=mergedcoverage`; on PowerShell use a profile path **without** a `.out` suffix, for example `mergedcoverage`, `ph`, or `sv`, so `-func=` is not misparsed)
- **Execution Time**: on the order of a few seconds for `go test ./...` on a typical dev machine
- **Reliability**: deterministic; use `go test -race ./...` where supported (see CI)

This foundational test suite provides a solid base for testing core functionality, with room for significant expansion in the main application components.
