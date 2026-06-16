---
name: plugins-marchat
description: >-
  Develops and maintains marchat plugins: SDK, host IPC, manager install path,
  store downloads, and Ed25519 licenses. Use when editing plugin/, plugin
  commands, or plugin_state.json behavior.
paths:
  - "plugin/**"
  - "cmd/license/**"
---

# Plugins (marchat)

JSON IPC over stdin/stdout. Packages: `plugin/sdk`, `plugin/host`, `plugin/manager`, `plugin/store`, `plugin/license`.

## SDK (`plugin/sdk`)

- Separate `go.mod`; root `go test ./...` does **not** run SDK tests.
- Stdio loop: `RunIO` / `HandlePluginRequest` (`stdio_test.go`).
- Run: `cd plugin/sdk && go test ./...`

## Host

- `StartPlugin` / `StopPlugin`; `StopPlugin` waits for stdout/stderr reader goroutines before niling pipes (race-safe disable/enable).
- Bounded plugin chat fan-out queue (`plugin/host/outbound_test.go`).
- Serialized stdin writes via `stdinMu` so fan-out and `ExecuteCommand` cannot interleave JSON lines.

## Manager

- State in `plugin_state.json` under server data directory.
- Install: SHA-256 checksum, size limits, zip-slip checks, staging rollback, execute bit on binary by exact name.
- `file://` URLs: `plugin/fileurl` (Linux/Windows).
- Non-admin users may run chat commands when manifest `AdminOnly: false`.

## Licenses

- Ed25519 signing/validation (`plugin/license/`, `cmd/license`).
- Cache entries re-signature-checked; plugin name must match cache key.
- Separate from chat E2E crypto.

## Server integration

- `server/plugin_commands.go` dispatches to manager.
- Hub stays off plugin IPC for core routing; plugin messages use bounded fan-out.

## Testing

- `plugin/integration_test.go`, `manager_lifecycle_test.go`, `plugin_lifecycle_test.go` (built minimal plugin).
- CI runs nested module fmt and govulncheck (see `go.yml`).

## Docs

- `PLUGIN_ECOSYSTEM.md`, `plugin/README.md` for author-facing detail.
