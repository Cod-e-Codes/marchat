---
name: database-marchat
description: >-
  Changes marchat SQL schema and queries across SQLite, PostgreSQL, and MySQL
  using dialect helpers. Use when editing server/db.go, db_dialect.go, schema
  migrations, MARCHAT_DB_PATH, or multi-database behavior.
paths:
  - "server/db.go"
  - "server/db_dialect.go"
  - "server/db_*_test.go"
  - "server/handlers.go"
  - "server/message_state.go"
---

# Database (marchat)

Runtime backend via `MARCHAT_DB_PATH`: SQLite (default), PostgreSQL, or MySQL. Detection and DSN parsing: `server/db_dialect.go`, `InitDB` in `server/db.go`.

## Rules

- Parameterized queries only; no string-concatenated user input.
- Every schema or query change must work on all three dialects (or use `db_dialect.go` helpers).
- SQLite: WAL when backend is SQLite; quote paths safely in `VACUUM INTO` and similar.
- MySQL: DSN via `mysql:` or `mysql://`; `mysql.Config` with `parseTime=true`; indexed text rules for search.
- Postgres: boolean columns need dialect boolean literals, not `= 0` / `= 1`.

## Durable tables

Include messages plus durable state: reactions, read receipts, `user_message_state`, channel preferences. Not message rows alone.

## Testing

| Level | Where |
|-------|--------|
| Unit / integration | In-memory or temp SQLite in `server/*_test.go` |
| CI smoke | `server/db_ci_smoke_test.go` with `MARCHAT_CI_POSTGRES_URL`, `MARCHAT_CI_MYSQL_URL` |
| Handlers | Visible replay SQL (`GetRecentMessagesForUser`), search, pin toggle |

Locally, CI smoke tests skip without env vars. See `testing-marchat` skill.

## Schema change workflow

1. Update `CreateSchema` / migrations in `db.go` with dialect branches.
2. Add or extend `db_dialect_test.go` for new SQL fragments.
3. Run `go test ./server/...`.
4. Document env or migration notes in `ARCHITECTURE.md` / `CHANGELOG.md` if user-visible.

## References

- `ARCHITECTURE.md` database section
- `internal/doctor` DB probes for `-doctor` output
