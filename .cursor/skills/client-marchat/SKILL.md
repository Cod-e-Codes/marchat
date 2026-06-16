---
name: client-marchat
description: >-
  Implements marchat terminal client TUI, commands, keystore, and WebSocket
  client behavior. Use when editing client/, Bubble Tea models, :commands,
  themes, notifications, or client configuration paths.
paths:
  - "client/**"
---

# Client (marchat)

Bubble Tea + Lipgloss TUI. Entry: `client/main.go`; split across `render.go`, `commands.go`, `hotkeys.go`, `websocket.go`, `cli_output.go`, `notification_manager.go`, etc.

## Patterns

- **Reconnect**: exponential backoff (capped); no reconnect on fatal username/handshake errors (`websocket.go`, `main.go`).
- **Commands**: `:q` quits; `Esc` closes menus; help in `commands.go` (shortcuts vs text commands).
- **E2E**: same wire path for channel text and DMs when encryption on; files via keystore `EncryptRaw` / `DecryptRaw`.
- **Config**: `MARCHAT_CONFIG_DIR` / `ResolveClientConfigDir()`; keystore path resolution and migration in `client/config/config.go`.
- **Chrome**: terminal-native labels; no decorative lock emoji in UI chrome (user content may include emoji).
- **Metadata**: Alt+M or `:msginfo` toggles message id and encrypted flag per line (`render.go`).
- **Notifications**: bell, desktop (Alt+N), `:notify-mode`, `:quiet`, `:focus` (`notification_manager.go`).
- **Pre-TUI output**: colorized stdout via `cli_output.go` unless `NO_COLOR`.

## Testing

- Inject `tea.Msg` in tests; no real terminal.
- `client/main_test.go`, `websocket_e2e_test.go`, `keystore_test.go`, `config_test.go`.
- See `testing-marchat` skill.

## Protocol

Outbound/inbound shapes must match `PROTOCOL.md` (`protocol-marchat` skill).

## Doctor

`go run ./client -doctor` reports profiles, keystore, E2E key source, `dm_state.json`.
