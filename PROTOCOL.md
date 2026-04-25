# marchat Protocol Specification

This document outlines the communication protocol used by `marchat`, a terminal-based chat application built with Go and Bubble Tea. It covers WebSocket interactions, message formats, and expected client behavior. The protocol is designed for simplicity, extensibility, and ease of implementation for alternative clients or integrations.

For optional graphical clients that implement this protocol, see [README.md](README.md#optional-graphical-clients).

---

## WebSocket Connection

Clients connect to the server via WebSocket:

```
/ws
```

The server may run over either `ws://` (HTTP) or `wss://` (HTTPS/TLS) depending on configuration. The connection scheme is determined by whether TLS certificates are provided to the server.

After a successful WebSocket upgrade, the client must immediately send a handshake message.

---

## Handshake

The handshake message introduces the user to the server. It must be the first message sent after connection.

### Format

```json
{
  "username": "alice",
  "admin": true,
  "admin_key": "your-admin-key"
}
```

### Fields

- `username` (string): **Required.** Display name. Must be unique among currently connected users. The server reserves the name under lock during handshake before registering the session, so two simultaneous connections cannot claim the same username.
- `admin` (bool): Optional. Request admin access. Defaults to `false`.
- `admin_key` (string): Required only if `admin` is `true`. Must match the server-configured key.

If `admin` is requested:

- The username must match one in the admin allowlist (case-insensitive).
- The provided key must match the server’s configured key.
- If the key does not match, the server sends a JSON message with `type` **`auth_failed`** and `data` containing `reason: "invalid admin key"`, then a **1008** WebSocket close with reason `Invalid admin key` (same RFC 6455 close frame format as other handshake rejections).

Invalid handshakes (missing username, duplicate names, or invalid admin credentials) result in immediate connection termination. The server sends a **standard WebSocket close frame** (RFC 6455): a registered **close code** plus an optional UTF-8 **reason** string (not raw text without a code). Handshake JSON parse failures use **1002** (`CloseProtocolError`); policy rejections (empty username, invalid username, allowlist, non-admin claiming admin, duplicate username, ban, etc.) use **1008** (`ClosePolicyViolation`). Duplicate usernames typically include reason text such as `Username already taken - please choose a different username`. Alternative clients should read the close code and reason from their WebSocket API, not assume the payload is plain text.

---

## Message Format

All messages exchanged after handshake use JSON.

### Chat Messages

```json
{
  "sender": "alice",
  "content": "hello world",
  "created_at": "2025-07-24T15:04:00Z",
  "type": "text",
  "channel": "general",
  "message_id": 42,
  "edited": false,
  "encrypted": false
}
```

Optional fields (`recipient`, `reaction`, etc.) are omitted from JSON when unset.

#### Fields

- `sender` (string): Username of the sender.
- `content` (string): Message text. Empty if type is `file`. For `search`, carries the query string.
- `created_at` (string): RFC3339 timestamp.
- `type` (string): Core types include `"text"`, `"file"`, and `"admin_command"`. See [Extended Message Types](#extended-message-types) for additional values.
- `file` (object, optional): Present only when `type` is `"file"`.
- `message_id` (int64, optional): Unique database identifier for a message; used for edits, deletes, reactions, and pins. Assigned by the server when a text message is persisted.
- `recipient` (string, optional): Target username for direct messages (`type`: `"dm"`). Empty for normal channel/broadcast traffic.
- `edited` (bool, optional): `true` if the message body was edited after send.
- `channel` (string, optional): Channel name for scoped delivery; defaults to `"general"` when omitted on outgoing chat messages. See [Channels](#channels).
- `reaction` (object, optional): Reaction payload when `type` is `"reaction"`. See [Reaction object](#reaction-object).
- `encrypted` (bool, optional): `true` when `content` is end-to-end encrypted (opaque to the server). See [End-to-end encryption](#end-to-end-encryption).

#### File Object

```json
{
  "filename": "screenshot.png",
  "size": 23456,
  "data": "<base64-encoded>"
}
```

Maximum file size is configurable (default 1MB). Files exceeding this size are rejected.
Configure via environment variables on the server:

- `MARCHAT_MAX_FILE_BYTES`: exact byte limit (takes precedence)
- `MARCHAT_MAX_FILE_MB`: size in megabytes

If neither is set, the default is 1MB.

#### Reaction object

When `type` is `"reaction"`, `reaction` must be set:

```json
{
  "emoji": "👍",
  "target_id": 42,
  "is_removal": false
}
```

- `emoji` (string): Emoji character or shorthand alias. The client resolves aliases before sending (e.g. `+1` becomes `👍`, `heart` becomes `❤️`). Raw emoji characters are also accepted. **Supported aliases**: `+1`, `-1`, `heart`, `laugh`, `fire`, `party`, `eyes`, `check`, `x`, `think`, `clap`, `rocket`, `wave`, `100`, `sad`, `wow`, `angry`, `skull`, `pray`, `star`.
- `target_id` (int64): `message_id` of the message being reacted to.
- `is_removal` (bool, optional): When `true`, removes the sender’s reaction instead of adding it.

### Extended Message Types

These values of `type` extend the core chat protocol:

| `type` | Purpose |
|--------|---------|
| `edit` | Replace the text of an existing message. Requires `message_id` and new body in `content`. Set `encrypted` to `true` when `content` is E2E ciphertext (same base64 **nonce ‖ ciphertext** layout as `text`); the server persists that flag to `is_encrypted` so history and reconnects stay consistent. Only the original sender may edit; admins cannot edit someone else's message (unlike `delete`, where an admin may remove any message). Enforced by matching WebSocket `sender` to the stored row. |
| `delete` | Soft-delete a message. Requires `message_id`. Authors may delete their own messages; admins may delete any message. |
| `typing` | Typing indicator; `content` is not required. The server sets `sender`. Optional `recipient` scopes typing to a DM partner only (same delivery as `dm`: sender and recipient). Empty `recipient` keeps channel/global behavior (see [Channels](#channels) when `channel` is set). The reference client sets `recipient` while composing in DM mode; it only applies DM-scoped typing when that DM thread is open, and it does not show channel/global typing while a DM thread is open (so channel typing is not mistaken for the active DM). |
| `reaction` | Add or remove a reaction. Requires the `reaction` object (`emoji`, `target_id`, optional `is_removal`). |
| `dm` | Direct message. Requires `recipient` (target username). Delivered only to sender and recipient. Persisted by the server and included in reconnect history only for those two participants. |
| `search` | Full-text search. `content` is the search query; the server replies with a private `text` message from `System` listing up to 20 matches (not broadcast). |
| `pin` | Toggle pinned state for `message_id`. **Admin only**; non-admins receive an error `text` from `System`. On success, a `System` `text` notice is broadcast. |
| `read_receipt` | Read receipt. Clients may send `read_receipt` with `message_id` set to the latest persisted chat line they have read while the transcript viewport is scrolled to the tail (the reference client debounces bursts). The server sets `sender` from the connection, persists when `message_id` is positive (`read_receipts` table), and broadcasts the message. Receivers may ignore or surface receipts in UI; the reference TUI does not print them in the transcript. |
| `join_channel` | Join a channel. Requires `channel` (name). If the client was in another channel, they leave it first. The server sends a confirmation `text` from `System`. |
| `leave_channel` | Leave the current channel and return to `#general`. No `content` required. If already in `general`, no-op. The server sends a confirmation `text` from `System`. |

### Server Events

Messages initiated by the server to update client state.

#### User List

```json
{
  "type": "userlist",
  "data": {
    "users": ["alice", "bob"]
  }
}
```

---

## End-to-end encryption

Optional chat encryption uses a **shared symmetric key** for all participants (global model), not per-user public-key exchange.

- **Key material**: 32 random bytes, distributed out-of-band. Clients load the same key via `MARCHAT_GLOBAL_E2E_KEY` (standard base64) or a locally generated key that operators then share manually. Persisted keys live in the client’s passphrase-protected **`keystore.dat`** (format and migration details: **README.md** / **ARCHITECTURE.md**). The reference client does **not** print the full auto-generated key to stdout (only a Key ID); use env, keystore copy, or another confidential channel. When the env var is set, it **wins** for that client process; the keystore file is unchanged unless the client saves a new key without the env var set.
- **Algorithm**: ChaCha20-Poly1305 (RFC 8439). Each encrypted payload uses a random **12-byte nonce** (typical for this AEAD).
- **Text messages on the wire**: Same JSON message shape as plaintext chat; set `encrypted` to `true`. The `content` field is **standard base64** encoding of **nonce ‖ ciphertext** (nonce first, 12 bytes, then the Poly1305-sealed ciphertext). The plaintext decrypted by the AEAD is a JSON object representing the inner chat message (e.g. sender, content, type, timestamp) as produced by the reference client.
- **Files**: When `type` is `file` and E2E is enabled, the reference client encrypts file bytes with the same global key; see client implementation for the exact binary layout (nonce-prefixed ciphertext).
- **Edits** (`type`: `edit`): When E2E is enabled, the reference client encrypts the new plaintext the same way as a normal `text` message and sets `encrypted` accordingly. The server stores the new opaque `content` and updates `is_encrypted` from the incoming `encrypted` field (it does not force plaintext).

The server stores and relays opaque `content` (and encrypted file blobs) without performing decryption.

---

## Server Behavior

- On successful handshake:
  - Sends up to 50 recent messages from history (newest first; clients typically display in chronological order). The same replay happens on every new WebSocket session after a disconnect. Clients that keep a local transcript in memory should **replace** or **dedupe** against that replay instead of appending it blindly, or they will show duplicate lines. DM history in this replay is scoped to the authenticated user (their sent DMs and DMs addressed to them). The reference TUI clears its transcript (messages, reactions, typing state, and cached received-file metadata) when a connection succeeds so the replay is the single source of truth for the visible scrollback window.
  - Sends current user list.
- On user connect/disconnect:
  - Broadcasts updated user list.
- On message send:
  - Persists eligible messages to the configured SQL backend selected by `MARCHAT_DB_PATH` (SQLite path, PostgreSQL DSN, or MySQL DSN).
  - Delivers to all connected clients **or** only to members of a channel when `channel` is non-empty and `sender` is not `System` (see [Channels](#channels)). Direct messages use a separate path (sender and recipient only).
- Reactions, read receipts, and last channel per user may be persisted server-side and replayed to reconnecting clients.
- DM unread counters and DM thread hide/archive state are client-side UI state in the reference TUI, not server protocol fields. The reference client stores this local state under its client config directory. Opening a DM thread marks that thread read immediately in the reference client.
- Message history is capped at 1000 messages.
- On successful `type`: `edit`, the server updates `content`, sets `edited`, and sets stored encryption metadata from the incoming `encrypted` field (so edited ciphertext rows remain ciphertext with `is_encrypted` aligned to the wire).
- `:`-prefixed input and `admin_command` messages go through the server command path (plugins first, then built-in admin commands for admins). Non-admins may receive a `System` `text` line when a command requires admin privileges. Admins receive a `System` `text` line when no plugin handler and no built-in handler matches the first token (content includes `Unknown command:` and that token).

---

## Channels

- Every connection is placed in the `general` channel after a successful handshake.
- Channel names are carried on messages via the optional `channel` field. When it is set on a non-system message, the server routes that message only to clients currently joined to that channel.
- Clients switch channels by sending `type`: `join_channel` with `channel` set to the target name. Sending `join_channel` while already in another channel leaves the previous channel first.
- Clients return to `general` with `type`: `leave_channel` (no fields required beyond the message envelope). If the client is already in `general`, the server performs no channel change.

---

## Rate Limiting

Per WebSocket connection, the server enforces rate limiting on **all** incoming JSON messages (including typing indicators and control types):

- **Burst:** at most **20 messages** per **5 second** sliding window.
- **Cooldown:** if the limit is exceeded, further incoming messages from that connection are ignored until **10 seconds** have elapsed since the violation, then counting resumes.

When the burst threshold is crossed, the server sends one `System` `text` notice to that connection, then ignores further incoming JSON until the cooldown elapses (the server still logs drops). Messages received during the cooldown window are dropped without an extra notice. Alternative clients should pace high-frequency traffic accordingly.

---

## Message Retention

- Messages are stored in the backend configured by `MARCHAT_DB_PATH` (SQLite/PostgreSQL/MySQL).
- The most recent 1000 messages are retained.
- Older messages are deleted automatically.

---

## Authentication

Admin status is optional and granted only if:

- `admin` is set to `true` in the handshake.
- `admin_key` matches the server’s configured key.
- `username` is on the allowed admin list, if configured.

If `admin` is not requested, `admin_key` is not required.

---

## Configuration

Client settings are usually stored in **`config.json`** under the **client configuration directory** (per-user application data, or `MARCHAT_CONFIG_DIR`). That directory is separate from the **server** directory (`.env` plus local DB file when SQLite is used). Use **`marchat-client -doctor`** / **`marchat-server -doctor`** to print resolved paths.

Example `config.json` shape:

```json
{
  "username": "YOUR_USERNAME",
  "server_url": "ws://localhost:9090/ws",
  "theme": "patriot",
  "twenty_four_hour": true
}
```

**Note**: The `server_url` should use `ws://` for HTTP connections or `wss://` for HTTPS/TLS connections, depending on the server's TLS configuration.

Sensitive values (like `admin_key`) are passed only during handshake and are not stored in config.

---

## Notes on Extensibility

The protocol is intentionally JSON-based. **Plugins** extend the server through a managed plugin host (separate processes, JSON IPC); chat messages with `type` `"text"` can be forwarded to plugins for automation, while `:`-prefixed lines and `admin_command` messages participate in the command pipeline (including built-in admin commands where authorized). The plugin SDK's `Message` struct mirrors the core wire fields (`sender`, `content`, `created_at`, `type`, `channel`, `encrypted`, `message_id`, `recipient`, `edited`) so plugins receive full message context. All extended fields use `omitempty`, making the wire format backwards-compatible with older compiled plugins. Clients and tools should use the `type` field and optional structured fields (`message_id`, `channel`, `reaction`, etc.) to interpret each payload.

---

## Example Workflow

1. Client connects via WebSocket to `/ws`
2. Sends handshake:

```json
{
  "username": "carol",
  "admin": false
}
```

3. Server responds with history and user list
4. Client sends message (optional `channel`; if omitted, the server uses the client’s current channel, default `general`):

```json
{
  "sender": "carol",
  "content": "hey there",
  "created_at": "2025-07-24T15:09:00Z",
  "type": "text",
  "channel": "general"
}
```

5. Server delivers the message to every connected client joined to that channel (clients in other channels do not receive it)

---

This document is intended to help developers build compatible clients, bots, or tools for `marchat`, or understand how the protocol works.

For questions or suggestions, please open a GitHub Discussion or Issue.
