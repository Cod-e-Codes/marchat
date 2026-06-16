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

- **Reconnect**: exponential backoff (capped at 30s); delay resets only after successful connect (`wsConnected`), not each `Init()` retry; no reconnect on fatal username/handshake errors (`websocket.go`, `main.go`).
- **Commands**: `:q` quits; `Esc` closes menus; help in `commands.go` (shortcuts vs text commands). Transient command results belong in the **banner** when short; longer lists (e.g. `:themes`) may use transcript System lines.
- **E2E**: same wire path for channel text and DMs when encryption on; files via keystore `EncryptRaw` / `DecryptRaw`. Do not log plaintext on send/decrypt paths.
- **Config**: `MARCHAT_CONFIG_DIR` / `ResolveClientConfigDir()`; keystore path resolution and migration in `client/config/config.go`.
- **Chrome**: terminal-native labels; no decorative lock emoji in UI chrome (user content may include emoji).
- **Metadata**: Alt+M or `:msginfo` toggles message id and encrypted flag per line (`render.go`).
- **Notifications**: bell, desktop (Alt+N), `:notify-mode`, `:quiet`, `:focus` (`notification_manager.go`).
- **Pre-TUI output**: colorized stdout via `cli_output.go` unless `NO_COLOR`.

## Transcript rendering (`render.go`)

- **Word wrap**: `wrapStyledBlock` + `ansi.Wrap` at viewport width; preserves ANSI codes.
- **URLs**: `prepareURLWrapping` (non-breaking hyphens in hosts), `markURLsForWrap` / `applyURLMarkers` so hyperlink color and underline survive line breaks without styling continuation indent. Click-to-open stitches wrapped lines, trims viewport padding, maps clicks via `buildTranscriptLineURLs` + `chatPanelOrigin()` (header, box border, banner), expands partial regex hits from message bodies; only the URL under the cursor opens.
- **Sort order**: `sortMessagesByTimestamp` / `messageLess` - persisted chat by `message_id`, server System (`message_id == 0`) by `created_at`, client-local System (negative `message_id`) after persisted chat.
- **Ephemeral System feedback**: `isTranscriptSystemNotice` / `isTranscriptSystemMessage` route command errors and one-line server replies (e.g. admin-only denial) to the **banner**; multi-line search, themes, and channel notices stay in the transcript. Negative `message_id` transcript notices are classified by **content** (`isTranscriptSystemNotice`), not ID sign alone. `pruneEphemeralSystemMessages` clears stale ephemeral lines on send or inbound persisted chat.
- **Client-local System lines**: negative `message_id` via `appendClientSystemMessage` for transcript notices only (with active `channel` set); short client usage/errors go to banner through `appendClientSystem`.

## Testing

- Inject `tea.Msg` in tests; no real terminal.
- `client/testmain_test.go`: `lipgloss.SetColorProfile(termenv.ANSI256)` so headless render/hyperlink tests emit real SGR sequences.
- `client/render_test.go`: URL wrap, hyperlink markers, system line severity, wrap width.
- `client/main_test.go`: DM/channel filters, unread, client system prune/sort, reconnect backoff, URL click hit/miss, E2E search hint.
- `client/websocket_e2e_test.go`, `keystore_test.go`, `config_test.go`.
- See `testing-marchat` skill.

## Protocol

Outbound/inbound shapes must match `PROTOCOL.md` (`protocol-marchat` skill). Negative `message_id` is client-only; never send on the wire.

## Doctor

`go run ./client -doctor` reports profiles, keystore, E2E key source, `dm_state.json`.
