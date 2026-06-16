---
name: developing-marchat
description: >-
  Implements and refactors marchat Go code with project toolchain and quality
  gates. Use when adding features, fixing bugs, touching client/server/plugin
  code, or when the user asks to develop, build, or validate marchat changes.
---

# Developing marchat

Read `.cursor/rules/marchat.mdc` for always-on constraints. Use domain skills (`client-marchat`, `server-marchat`, `plugins-marchat`, `database-marchat`, `protocol-marchat`) when work is localized.

## Before coding

1. Confirm scope: client, server, shared wire types, plugins, config, or docs-only.
2. Read the matching domain skill (`client-marchat`, etc.) **before** editing that area.
3. Check `ARCHITECTURE.md` and `PROTOCOL.md` if the change crosses process boundaries.
4. Prefer extending existing types and helpers over new parallel APIs.
5. Verify library APIs and toolchain from `go.mod`, repo source, or official docs - not model memory.

## Current information

This repo (`go.mod`, `ARCHITECTURE.md`, `PROTOCOL.md`, `.cursor/`) overrides general model knowledge. For Cursor skills/rules, CI, packaging, or dependency APIs not spelled out here, web search or fetch official documentation before implementing.

## Implementation rules

- Go 1.25+ idioms; toolchain patch in `go.mod` is authoritative for CI and Docker.
- Never hand-edit `go.mod` versions; use `go get -u package` or `go get package@latest`, then `go mod tidy`.
- Parameterized SQL only; dialect differences go through `server/db_dialect.go`.
- Chat E2E is a global ChaCha20-Poly1305 symmetric key, not per-user X25519 exchange.
- Do not log secrets (keys, passphrases, admin keys, session secrets).
- Minimize diff scope; no drive-by refactors or stubs.

## After substantive code changes

Run from repo root unless only `plugin/sdk` changed:

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go test -coverprofile=mergedcoverage ./...
go tool cover -func=mergedcoverage
```

If `golangci-lint` is installed:

```bash
golangci-lint run ./...
```

Nested `plugin/sdk` module (separate `go.mod`): run on every substantive root change, and always when `plugin/sdk` files changed:

```bash
cd plugin/sdk && go test ./...
```

Windows: CI runs `-race` on Linux; local `-race` on Windows may require CGO for some packages - still run it when the toolchain allows. See `testing-marchat` for coverage profiles and CI DB smoke.

## Completion checklist (all required before responding)

- [ ] Compiles; `go vet` clean for touched packages
- [ ] `go test ./...` and `go test -race ./...` run and reported (not assumed)
- [ ] `cd plugin/sdk && go test ./...` run (nested module; skip only for docs-only or unrelated isolated edits)
- [ ] Tests added or updated for behavior changes (`testing-marchat` skill)
- [ ] Docs/changelog/coverage updated if user-visible or totals shifted (`writing-marchat-docs`); update ARCHITECTURE/PROTOCOL/skills when behavior changes, not CHANGELOG alone
- [ ] Domain skill and `.cursor/skills/` updated when shipped behavior or agent workflow changes
- [ ] No protocol or keystore breaking change without explicit discussion and changelog note
- [ ] **Commit message drafted** via `git-workflow-marchat` for **all** uncommitted files in the working tree

## Git

Do not `git commit` or `git push` unless the user asks. **Always** end substantive work with a suggested commit message covering the full `git diff`, not only files touched in the last reply.
