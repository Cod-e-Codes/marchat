---
name: database-marchat
description: >-
  Changes marchat SQL schema and queries across SQLite, PostgreSQL, and MySQL
  using dialect helpers. Use when editing server/db.go, db_dialect.go, schema
  migrations, MARCHAT_DB_PATH, or multi-database behavior.
paths:
  - "server/db.go"
  - "server/db_dialect.go"
  - "server/migrate.go"
  - "server/db_*_test.go"
  - "server/migrate_test.go"
  - "server/handlers.go"
  - "server/message_state.go"
---

# Database (marchat)

Runtime backend via `MARCHAT_DB_PATH`: SQLite (default), PostgreSQL, or MySQL. Detection and DSN parsing: `server/db_dialect.go`, `InitDB` in `server/db.go`.

## Rules

- Parameterized queries only; no string-concatenated user input.
- Every schema or query change must work on all three dialects (or use `db_dialect.go` helpers).
- SQLite (`InitDB` only):
  - Put connection pragmas in the DSN so every pooled connection gets them (`_busy_timeout=5000`, `_journal_mode=WAL`, `_synchronous=NORMAL`, plus `_pragma` for cache/temp). Join with `?` or `&` if the path already has a query (`appendSQLiteDSNPragmas`).
  - After `Ping`, set `SetMaxOpenConns(1)` and `SetMaxIdleConns(1)`. Do **not** leave the default multi-connection `database/sql` pool on SQLite.
  - Do **not** rely on one-shot `Exec("PRAGMA ...")` after open for settings that must stick on every connection (that was the #118 `SQLITE_BUSY` failure mode).
  - Verify after open: `busy_timeout > 0`; for file-backed DBs, `journal_mode` is `wal`. In-memory (`:memory:` / `mode=memory`) requires busy_timeout only.
  - Quote paths safely in `VACUUM INTO` and similar.
- Postgres/MySQL: leave pool defaults alone (do not force `MaxOpenConns(1)`).
- MySQL: DSN via `mysql:` or `mysql://`; `mysql.Config` with `parseTime=true`; indexed text rules for search.
- Postgres: boolean columns need dialect boolean literals, not `= 0` / `= 1`.

## Durable tables

Include messages plus durable state: reactions, read receipts, `user_message_state`, channel preferences. Not message rows alone.

## Testing

| Level | Where |
|-------|--------|
| Unit / integration | In-memory or temp SQLite in `server/*_test.go` (`db_test.go` covers DSN join, file WAL, `:memory:`, concurrent inserts) |
| CI smoke | `server/db_ci_smoke_test.go` with `MARCHAT_CI_POSTGRES_URL`, `MARCHAT_CI_MYSQL_URL` (also asserts pool is not forced to 1) |
| Handlers | Visible replay SQL (`GetRecentMessagesForUser`), search, pin toggle |

Locally, CI smoke tests skip without env vars. See `testing-marchat` skill.

## Schema change workflow

1. Add a new migration step in `server/migrate.go` (`applyMigrationV2`, etc.) and bump `currentSchemaVersion`; extend `verifySchema` when new required tables or columns ship. Prefer deterministic DDL for versions after the v1 baseline (avoid inspect-and-reconcile).
2. `MigrateSchema` runs ordered migrations, records `schema_version`, and verifies required tables (including `ban_history.expires_at`). SQLite/Postgres wrap each version in a transaction; MySQL cannot (DDL implicit commit) - document that when changing migrator behavior. `CreateSchema` in the same file is a thin `log.Fatal` wrapper for tests.
3. Add or extend `db_dialect_test.go` for new SQL fragments.
4. Run `go test ./server/...`.
5. Document env or migration notes in `ARCHITECTURE.md` / `CHANGELOG.md` if user-visible.

## Backup

In-process `:backup` supports **SQLite only**. Postgres and MySQL require native backup tools; `BackupDatabase` returns an error when dialect is not SQLite.

## References

- `ARCHITECTURE.md` database section
- `internal/doctor` DB probes for `-doctor` output
