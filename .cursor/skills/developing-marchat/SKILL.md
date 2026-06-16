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
2. Check `ARCHITECTURE.md` and `PROTOCOL.md` if the change crosses process boundaries.
3. Prefer extending existing types and helpers over new parallel APIs.

## Current information

This repo (`go.mod`, `ARCHITECTURE.md`, `PROTOCOL.md`, `.cursor/`) overrides general model knowledge. For Cursor skills/rules, CI, packaging, or library APIs not spelled out here, web search or fetch official documentation before implementing.

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
```

If `golangci-lint` is installed:

```bash
golangci-lint run ./...
```

Nested module when `plugin/sdk` changed:

```bash
cd plugin/sdk && go test ./...
```

## Completion checklist

- [ ] Compiles; `go vet` clean for touched packages
- [ ] Tests added or updated for behavior changes (`testing-marchat` skill)
- [ ] Docs/changelog updated if user-visible (`writing-marchat-docs` skill)
- [ ] No protocol or keystore breaking change without explicit discussion and changelog note

## Git

Do not `git commit` or `git push` unless the user asks. Offer a commit message via `git-workflow-marchat` when appropriate.
