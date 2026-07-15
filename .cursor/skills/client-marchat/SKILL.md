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

Bubble Tea + Lipgloss TUI on **Charm v2** (`charm.land/bubbletea/v2`, `bubbles/v2`, `lipgloss/v2`). Entry: `client/main.go`; split across `render.go`, `commands.go`, `hotkeys.go`, `websocket.go`, `scroll_input.go`, `cli_output.go`, `notification_manager.go`, etc.

## Patterns

- **Charm v2**: `newMainTeaView` sets `AltScreen`, `MouseModeCellMotion` (disabled while Shift is held for terminal drag-select), and `tea.View.BackgroundColor` (`altScreenFill` / black). `chromeComposerPanel` is full-width with theme `Input` background only (textarea styles are foreground-only). Transcript interior uses `transcriptFill` on `Box`. `configureTextareaChrome` syncs textarea colors with theme `Input`. Multiline composer: Ctrl+J via `textarea.Update`; up/down move cursor when value contains `\n`, else scroll chat. `KeyPressMsg` / `KeyReleaseMsg`; bubbles use `SetWidth` / `SetHeight` / `SetStyles`.
- **Reconnect**: exponential backoff (capped at 30s); delay resets only after successful connect (`wsConnected`), not each `Init()` retry; no reconnect on fatal username/handshake errors (`websocket.go`, `main.go`).
- **Commands**: `:q` quits; `Esc` closes menus; help in `commands.go` (shortcuts vs text commands). Transient command results belong in the **banner** when short; longer lists (e.g. `:themes`) may use transcript System lines.
- **E2E**: same wire path for channel text and DMs when encryption on; files via keystore `EncryptRaw` / `DecryptRaw`. Do not log plaintext on send/decrypt paths.
- **Config**: `MARCHAT_CONFIG_DIR` / `ResolveClientConfigDir()`; keystore path resolution and migration in `client/config/config.go`.
- **Chrome**: terminal-native labels; no decorative lock emoji in UI chrome (user content may include emoji).
- **Metadata**: Alt+M or `:msginfo` toggles message id and encrypted flag per line (`render.go`).
- **Notifications**: bell, desktop (Alt+N), `:notify-mode`, `:quiet`, `:focus` (`notification_manager.go`, `notification_desktop.go`). Windows: Go-built toast XML + `powershell -EncodedCommand`; macOS: `strconv.Quote` for `osascript -e`; Linux: `notify-send` argv args.
- **Pre-TUI output**: colorized stdout via `cli_output.go` unless `NO_COLOR`.

## Transcript rendering (`render.go`)

- **Word wrap**: `wrapStyledBlock` + `ansi.Wrap` at viewport width; preserves ANSI codes.
- **URLs**: `prepareURLWrapping` (non-breaking hyphens in hosts), `markURLsForWrap` / `applyURLMarkers` so hyperlink color, underline, and OSC 8 sequences survive line breaks with the **full** href on every wrapped fragment. Mouse click-to-open (`findURLAtClickPosition`) remains as fallback when OSC 8 is unavailable. Copy from message still works everywhere.
- **Scroll input**: `activeScrollViewport` routes keyboard and `MouseWheelMsg` to help, DB menu, chat, or user list viewports; list modals (file picker, code-snippet language) handle wheel via `CursorUp`/`CursorDown`. Help and DB menu overlays (`overlayCapturesKeyboard`) swallow non-scroll keys so chat typing indicators, URL clicks, and read-receipt flush do not fire while browsing overlay content; `maybeFlushReadReceipt` only runs when the chat transcript viewport is active and at bottom.
- **Sort order**: `sortMessagesByTimestamp` / `messageLess` - persisted chat by `message_id`, server System (`message_id == 0`) by `created_at`, client-local System (negative `message_id`) after persisted chat.
- **Ephemeral System feedback**: `isTranscriptSystemNotice` / `isTranscriptSystemMessage` route command errors and one-line server replies (e.g. admin-only denial) to the **banner**; multi-line search, themes, and channel notices stay in the transcript. Negative `message_id` transcript notices are classified by **content** (`isTranscriptSystemNotice`), not ID sign alone. `pruneEphemeralSystemMessages` clears stale ephemeral lines on send or inbound persisted chat.
- **Client-local System lines**: negative `message_id` via `appendClientSystemMessage` for transcript notices only (with active `channel` set); short client usage/errors go to banner through `appendClientSystem`.

## Testing

- Inject `tea.Msg` in tests; no real terminal.
- `client/testmain_test.go`: `lipgloss.Writer.Profile = colorprofile.ANSI256` so headless render/hyperlink tests emit real SGR and OSC 8 sequences.
- `client/render_test.go`: URL wrap, OSC 8 hyperlink markers (`\x1b]8;;`), underline on wrapped segments, system line severity, wrap width; URL click helpers (`buildTranscriptLineURLs`, `findURLAtTranscriptClick`) are headless fallback-path coverage.
- `client/main_test.go`: DM/channel filters, unread, client system prune/sort, reconnect backoff, URL click hit/miss (single-line, headless), E2E search hint.
- `client/scroll_input_test.go`: scroll target selection, help viewport wheel handling, overlay input capture, read-receipt scoping, viewport dimension helpers
- `client/websocket_e2e_test.go`, `keystore_test.go`, `config_test.go`, `notification_manager_test.go` (desktop notification escaping).
- See `testing-marchat` skill.

## Protocol

Outbound/inbound shapes must match `PROTOCOL.md` (`protocol-marchat` skill). Negative `message_id` is client-only; never send on the wire.

## Doctor

`go run ./client -doctor` reports profiles, keystore, E2E key source, `dm_state.json`.
