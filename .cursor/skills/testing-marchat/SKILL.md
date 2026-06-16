---
name: testing-marchat
description: >-
  Writes and runs marchat tests with race detection, coverage, and dialect smoke
  patterns. Use when adding tests, fixing test failures, measuring coverage,
  or when the user mentions go test, -race, TESTING.md, or CI database smoke.
paths:
  - "**/*_test.go"
  - "TESTING.md"
---

# Testing marchat

Authoritative tables and file list: `TESTING.md`. Do not weaken tests to make CI pass.

## Commands

Main module (repo root). `developing-marchat` requires both plain and race runs after substantive changes:

```bash
go test ./...
go test -race ./...
go test -coverprofile=mergedcoverage ./...
go tool cover -func=mergedcoverage
```

Nested SDK (separate `go.mod`; not in root merged coverage):

```bash
cd plugin/sdk && go test -coverprofile=sdkcover ./... && go tool cover -func=sdkcover
```

On Windows PowerShell, prefer a cover profile name without a `.out` suffix (e.g. `mergedcoverage`, `coverage`).

Stale cache:

```bash
go clean -cache -testcache
```

## Patterns

| Area | Pattern |
|------|---------|
| SQL / handlers | In-memory SQLite; table-driven cases in `server/*_test.go` |
| Postgres / MySQL | `server/db_ci_smoke_test.go`; skip locally unless `MARCHAT_CI_POSTGRES_URL` / `MARCHAT_CI_MYSQL_URL` set; MySQL DSN uses `mysql:` or `mysql://` prefix |
| Doctor env | `internal/doctor/env.go`: swap `osEnviron` under `environMu`; no `t.Parallel()` with other tests that swap it |
| TUI client | Inject `tea.Msg`; no real terminal; `testmain_test.go` sets Lipgloss ANSI256 for SGR assertions |
| Client render | `render_test.go`: URL wrap/hyperlink; `main_test.go`: transcript sort, prune, filters |
| cmd/server | Subprocess pattern in `cmd/server/subprocess_doctor_test.go` |
| Plugins | `plugin/host/plugin_lifecycle_test.go` builds minimal plugin with `go build` |
| Race | CI runs `go test -race ./...`; plugin host `StopPlugin` must drain readers before reuse |

Set `MARCHAT_DOCTOR_NO_NETWORK=1` in tests that hit doctor update checks.

## Coverage

- Main module overall statement coverage: see `README.md` and `TESTING.md` (regenerate; do not guess).
- `plugin/sdk`: measure separately (see `TESTING.md`).
- After material shifts (about 0.2% or new test files), update figures in `README.md` and `TESTING.md`.

## Anti-patterns (never do these)

- Skipping `-race` or nested `plugin/sdk` module when relevant code changed
- `time.Sleep` for synchronization without strong justification
- Fudging assertions, deleting tests, or conditional passes (`if os.Getenv(...) { t.Skip() }` without CI contract) to green CI
- Asserting only substring presence when ordering or side effects matter
- Schema changes tested only on SQLite when dialect SQL diverges
- Claiming tests pass without running `go test` on touched packages

## Checklist

- [ ] New behavior has a focused test
- [ ] `go test ./...` passes; `-race` when touching concurrency
- [ ] `plugin/sdk` tested if SDK touched
- [ ] Coverage docs updated if totals moved materially
- [ ] Report actual `go test` outcome in the final response
