# Changelog

Narrative notes by release. Per-file binaries and assets: [GitHub releases](https://github.com/Cod-e-Codes/marchat/releases).

## v1.0.0 (current)

**Released 2026-04-17.** Since **[v0.11.0-beta.5](https://github.com/Cod-e-Codes/marchat/releases/tag/v0.11.0-beta.5)**; compare [`v0.11.0-beta.5...v1.0.0`](https://github.com/Cod-e-Codes/marchat/compare/v0.11.0-beta.5...v1.0.0). Commits: **`git log v0.11.0-beta.5..v1.0.0 --oneline`**.

- **Client**: Terminal-native **footer and banner** chrome; **read receipts** in the transcript; **reconnect** clears stale transcript state; **sending** indicator and **unread** count refinements; **rate limit** notice when the server throttles; theme loader updates and **THEMES.md** examples.
- **Server**: Clearer handling for unknown **admin** commands over the admin connection; related client/server **sending-state** fixes after chat writes.
- **Docs / protocol**: **ARCHITECTURE**, **PROTOCOL**, **README**, **TESTING** aligned with TUI behavior and coverage.
- **Packaging**: **v1.0.0** templates across Homebrew, Scoop, winget, Chocolatey, and AUR; **Chocolatey** nuspec **iconUrl** (repo logo on `main`) and clearer **title**; refresh **zip SHA256** values from published release assets before `choco pack` / local manifest validation (see **PACKAGING.md**).

## v0.11.0-beta.5

**Released 2026-04-10.** Since **[v0.11.0-beta.4](https://github.com/Cod-e-Codes/marchat/releases/tag/v0.11.0-beta.4)**; compare [`v0.11.0-beta.4...v0.11.0-beta.5`](https://github.com/Cod-e-Codes/marchat/compare/v0.11.0-beta.4...v0.11.0-beta.5). Commits: **`git log v0.11.0-beta.4..v0.11.0-beta.5 --oneline`**.

- **Server**: RFC 6455 WebSocket close frames on handshake errors; hub stays off plugin IPC with bounded, best-effort, at-most-once plugin chat fan-out.
- **Client**: Experimental env-driven **exthook** and **`-doctor`** integration.
- **Plugin SDK**: **`RunStdio`** / **`HandlePluginRequest`** stdio loop; echo sample uses the SDK; docs and **README** plugin examples aligned (**GetConfig**, **Marshal**); **`plugin/sdk/cov`** gitignored; CI runs nested plugin modules (**fmt**, **govulncheck**).
- **Tests / CI**: Server loadverify benches and rate-limit coverage; **`-doctor`** tests use the injectable **`osEnviron`** hook under **`environMu`** (no parallel **`buildEnvLines`** tests that swap it); **plugin host** **`StopPlugin`** waits for stdout/stderr reader goroutines before reuse so **`-race`** is clean on disable/enable; Dependabot Node 20 note in **`.github/dependabot.yml`**.
- **Docs**: **TESTING** bench section; coverage/LoC tables refreshed from **`go test -coverprofile=mergedcoverage ./...`**; hook example lives under **`_example_hook`**; prose uses ASCII hyphens where edited.
- **Deps**: **`golang.org/x/crypto`**, **`golang.org/x/term`**, **`modernc.org/sqlite`**.

## v0.11.0-beta.4

**Released 2026-04-09.** [Compare from beta.3](https://github.com/Cod-e-Codes/marchat/compare/v0.11.0-beta.3...v0.11.0-beta.4). E2E edit consistency; deterministic theme cycle; security scanner vs **govulncheck** docs; **`.gitattributes`** LF normalization.

## v0.11.0-beta.3

**Released 2026-04-09.** [Compare from beta.2](https://github.com/Cod-e-Codes/marchat/compare/v0.11.0-beta.2...v0.11.0-beta.3). Keystore v3 and config/path fixes; web admin refresh; plugin SDK context and host fixes; DB smoke CI; Go 1.25.9; demos, E2E docs, and release asset workflow updates.

## v0.11.0-beta.2

Go 1.25.8 toolchain/docs; **`-doctor`** and env reflection improvements; terminal chrome and **`:msginfo`** metadata; license cache and server hardening; static release zips + **linux-arm64** for Termux.

## Earlier

- **v0.11.0-beta.1**: Multi-DB (SQLite / Postgres / MySQL), reactions, read receipts, message state, serialized WS writes, admin TUI ([PR #83](https://github.com/Cod-e-Codes/marchat/pull/83)).
- **v0.10.x**: Core chat features (edit/delete/pin/search, DMs, channels, E2E files, plugins), **`-doctor`**, Docker, Caddy TLS proxy docs ([**deploy/CADDY-REVERSE-PROXY.md**](deploy/CADDY-REVERSE-PROXY.md)), **`config/.env`** precedence.
