---
name: protocol-marchat
description: >-
  Maintains marchat WebSocket JSON protocol and shared wire types including E2E
  encoding. Use when changing PROTOCOL.md, shared/types.go, shared/crypto.go,
  handshake, message types, or client-server wire compatibility.
paths:
  - "PROTOCOL.md"
  - "shared/**"
  - "client/websocket*.go"
---

# Protocol (marchat)

Normative spec: `PROTOCOL.md`. Types and crypto: `shared/types.go`, `shared/crypto.go`.

## Wire basics

- WebSocket `/ws`; JSON messages.
- Handshake is first message after connect (`username`, optional `admin` / `admin_key`).
- Message types: text, system, typing, reactions, edits, deletes, pins, DMs (with `recipient`), channels, read receipts, files.

## E2E (chat)

- **Global shared symmetric key** (32 bytes), ChaCha20-Poly1305 on the wire.
- Server stores opaque ciphertext; does not decrypt chat content.
- When `encrypted` is true, `content` is base64-encoded `nonce || ciphertext`.
- Key distribution: out-of-band (`MARCHAT_GLOBAL_E2E_KEY`, or shared `keystore.dat` + passphrase).
- **Not** per-user X25519 key exchange for chat.

Client keystore: `client/crypto/keystore.go` (PBKDF2 + AES-GCM; v3 portable salt; legacy migrate on load).

## Breaking changes

- Avoid breaking handshake or message JSON without discussion.
- Document in `CHANGELOG.md` and `PROTOCOL.md`.
- Consider older clients on mixed-version servers.

## Change workflow

1. Update `shared/` types and constants first.
2. Server hub/handlers relay or persist without decrypting E2E payloads.
3. Client encode/decode in `client/websocket.go` and related paths.
4. Add round-trip tests (`client/websocket_e2e_test.go`, `shared/crypto_test.go`).
5. Sync `PROTOCOL.md` and `ARCHITECTURE.md`.

Plugin wire types extend `plugin/sdk`; keep compatible with `shared` message shapes where bridged.
