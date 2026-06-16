---
name: debugging-marchat
description: >-
  Diagnoses marchat client and server issues using -doctor, env configuration,
  and logs. Use when debugging connection failures, E2E decryption, database
  DSN, handshake errors, or when the user mentions doctor, diagnostics, or
  MARCHAT_ environment variables.
---

# Debugging marchat

## Doctor (first step)

| Binary | Command |
|--------|---------|
| Client | `go run ./client -doctor` or `marchat-client -doctor` |
| Server | `go run ./cmd/server -doctor` or `marchat-server -doctor` |
| JSON | Add `-doctor-json` (plain JSON; no TTY colors) |

Server doctor lists `MARCHAT_*` after `config/.env` load (`godotenv.Overload`: file wins on same key).

Skip GitHub release compare in tests or offline:

```bash
export MARCHAT_DOCTOR_NO_NETWORK=1
```

Implementation: `internal/doctor/`. DB dialect and DSN shape checks live there and in `server/db_dialect.go`.

## Common issues

| Symptom | Check |
|---------|--------|
| Handshake rejected / username taken | Reserved names; duplicate connection; server hub lock |
| WSS behind proxy | `deploy/`, `docker-compose.proxy.yml`, README proxy section; client WSS URL |
| E2E decrypt fails | Same global key on all clients (`MARCHAT_GLOBAL_E2E_KEY` or shared `keystore.dat`); not X25519 per-user |
| Postgres boolean errors | Dialect boolean helpers in `server/db_dialect.go` (`:search`, pin toggle) |
| MySQL time parsing | `mysql.Config` with `parseTime=true` in `InitDB` |
| SQLite path vs remote DSN | `MARCHAT_DB_PATH`; `mysql:` / `postgres:` prefixes for driver detection |
| Plugin disable race | `StopPlugin` waits for stdout/stderr readers (`plugin/host`) |
| Rate limit | `server/loadverify_ratelimit_test.go` constants match `client.go` read pump |

## WebSocket

- Serialized writes per connection (`server/client.go`).
- Origin checks use parsed hostnames; optional `MARCHAT_ALLOWED_ORIGINS`.
- Trusted proxy headers: `MARCHAT_TRUSTED_PROXIES` for `X-Forwarded-For` / `X-Real-IP`.

## Logging

Never log passphrases, admin keys, session secrets, or raw E2E keys.

## Workflow

1. Reproduce with minimal steps.
2. Run `-doctor` on client and server.
3. Read relevant test file for expected behavior before changing production code.
4. Fix root cause; add a regression test (`testing-marchat`).
