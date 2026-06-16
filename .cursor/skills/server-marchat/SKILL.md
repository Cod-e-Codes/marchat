---
name: server-marchat
description: >-
  Implements marchat server hub, WebSocket handlers, admin web, health, and
  startup validation. Use when editing server/, cmd/server/, hub routing, admin
  panel, or server configuration.
paths:
  - "server/**"
  - "cmd/server/**"
  - "config/**"
---

# Server (marchat)

App entry: `cmd/server/main.go`. Library: `server/` (hub, client, handlers, db, admin, health).

## Startup

- Validate before serve: at least one admin, non-empty admin key, valid listen port.
- Admin names: trim, lowercase, case-insensitive dedupe.

## Hub and WebSocket

- Per-channel routing, DMs, typing, read receipts, reactions.
- Outbound client messages are channel-stamped from hub membership (`stampClientChannel`); client-supplied `channel` values are ignored for routing.
- Reserved usernames during handshake (no double-book before registration).
- Serialized writes per connection (`client.go`).
- Read-pump rate limits: constants shared with `loadverify_ratelimit_test.go`.
- Handshake replay: up to 50 **visible** recent messages (SQL limit after DM/public filter).

## Admin

- TUI: `admin_panel.go`, `config_ui.go`.
- Web: `admin_web.go`, `admin_web.html`; `MARCHAT_SESSION_SECRET` (preferred), `MARCHAT_JWT_SECRET` deprecated; CSRF on mutating routes; login rate limit per IP.
- Trusted proxies: `MARCHAT_TRUSTED_PROXIES` for forwarded client IP.

## Security

- Origin checks on parsed hostnames; optional `MARCHAT_ALLOWED_ORIGINS`.
- Never log session secrets or admin keys.
- E2E payloads opaque at rest and in logs.

## Backup

- `:backup` and admin backup actions: SQLite only (`BackupDatabase` checks dialect). Postgres/MySQL return a clear error; use native tools for those backends.

## Config

- Root `config/` package: env + `config/.env` (`godotenv.Overload`).
- `MARCHAT_DB_PATH` for database backend (`database-marchat` skill).

## Testing

- `handlers_test.go`, `hub_test.go`, `integration_test.go`, `admin_web_test.go`.
- Subprocess doctor: `cmd/server/subprocess_doctor_test.go`.

## Health

Metrics and health HTTP: `server/health.go`.
