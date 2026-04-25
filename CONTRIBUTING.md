# Contributing to marchat

Thank you for your interest in contributing. This guide explains how to contribute effectively and what CI expects.

## Types of contributions

### Bug reports (GitHub Issues)

- Use the issue tracker for bugs and regressions.
- Include: clear description, steps to reproduce, expected vs actual behavior, OS and marchat version, and relevant logs or screenshots.

### Ideas and questions (GitHub Discussions)

- Use discussions for feature ideas, setup questions, feedback, and show-and-tell.

### Code contributions

1. **Fork and clone** the repo; keep your fork in sync with `main`.
2. **Branch from `main`** with a descriptive name; one logical change per PR.
3. **Read before you hack (suggested order):** [QUICKSTART.md](QUICKSTART.md) (mental model), [ARCHITECTURE.md](ARCHITECTURE.md) (components and data flow), [PROTOCOL.md](PROTOCOL.md) if you touch wire types or handlers, then [TESTING.md](TESTING.md) before you add tests.
4. **Implement** with focused diffs; match existing style and patterns in the packages you touch.
5. **Open a PR** against `main` with a clear description and links to issues.

## Local checks (match CI)

From the repo root, with Go 1.25.9+, `golangci-lint`, and `govulncheck` on your `PATH`:

```bash
gofmt -w .
golangci-lint run ./...
govulncheck ./...
go vet ./...
go test -race ./...
```

CI runs `gofmt -l .` and fails if it prints any path. After `gofmt -w .`, run `gofmt -l .` yourself and expect no output. Nested modules are **not** included in root `./...`:

```bash
cd plugin/sdk
go mod tidy
gofmt -w .
golangci-lint run ./...
govulncheck ./...
go vet ./...
go test -race ./...
cd ../examples/echo
go mod tidy
gofmt -w .
golangci-lint run ./...
govulncheck ./...
go vet ./...
go test -race ./...
cd ../../..
```

On Windows, run the same commands from `cmd` or PowerShell. For coverage profiles, see [TESTING.md](TESTING.md) (use a profile filename **without** a `.out` suffix in PowerShell so `go tool cover -func=` parses correctly).

## Code style

- Run `gofmt -w .` (or `gofmt -w` on specific paths); follow idiomatic Go.
- Fix new warnings from `go vet` and `golangci-lint`.
- Prefer table-driven tests where they clarify cases; avoid `t.Parallel()` in tests that swap global doctor env hooks (see [TESTING.md](TESTING.md)).

## Testing

- Add or extend tests for behavior you change; run the full root suite and nested modules as above.
- For coverage and package-level metrics, see [TESTING.md](TESTING.md).

## Automation

- GitHub Actions runs CI on PRs; tests and linters must pass.
- Dependabot proposes dependency updates; do not bump `go.mod` manually for routine upgrades unless you are fixing a specific issue.

## Communication

- Be respectful and constructive.
- Follow the [Code of Conduct](CODE_OF_CONDUCT.md).

Thank you for helping improve marchat.
