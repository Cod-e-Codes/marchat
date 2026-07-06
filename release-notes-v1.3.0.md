# marchat v1.3.0

## Client

- ANSI-aware word-wrap for chat bodies; reaction aliases `thumbsup` / `thumbsdown` and `:unreact`, `:thumbsup`, `:thumbsdown`
- When E2E is on and server search returns no matches, a `System` line notes ciphertext-only matching
- **Charm v2:** Bubble Tea, Bubbles, and Lip Gloss on `charm.land/*/v2`
- Long URLs wrap at path boundaries with OSC 8 hyperlinks; mouse click-to-open fallback ([#103](https://github.com/Cod-e-Codes/marchat/issues/103))
- **Fixes:** composer chrome, multiline keys, scroll-to-tail, mouse wheel routing, overlay input suppression, reconnect backoff, channel-scoped transcript/reactions/read receipts, no plaintext in E2E logs

## Server

- Replay up to 50 visible messages on every handshake (including reconnect)
- **Fixes:** Postgres boolean SQL, MySQL `parseTime=true`, channel-stamped outbound messages, channel-scoped typing/reactions/read receipts, SQLite-only `:backup`, admin TUI mouse scroll, `:cleardb` clears `user_message_state`

## Plugins

- **Fix:** serialized plugin stdin writes so chat fan-out and command RPC cannot corrupt IPC lines

## Dependencies

- charm.land/bubbletea/v2 v2.0.8, bubbles/v2 v2.1.0, lipgloss/v2 v2.0.5 (replaces Charm v1)
- golang.org/x/crypto v0.53.0, jackc/pgx/v5 v5.10.0, modernc.org/sqlite v1.53.0

Full narrative: [CHANGELOG.md](https://github.com/Cod-e-Codes/marchat/blob/main/CHANGELOG.md#v130)
