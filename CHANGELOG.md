# Changelog

Narrative notes by release. Per-file binaries and assets: [GitHub releases](https://github.com/Cod-e-Codes/marchat/releases).

## Unreleased

On **`main`** only; not part of the latest tagged release until you tag and publish. Compare against the current tag on [GitHub releases](https://github.com/Cod-e-Codes/marchat/releases).

- **Client**: Word-wrap chat message bodies to the transcript viewport width (ANSI-aware). Reaction aliases `thumbsup` / `thumbsdown`; `:unreact`, `:thumbsup`, and `:thumbsdown` commands. When E2E is on and server search returns no matches, a `System` line explains that search matches stored ciphertext, not decrypted plaintext.
- **Server**: Handshake replay queries up to 50 **visible** recent messages (SQL `LIMIT` after DM/public filter), on every connect including reconnect with no new traffic. `user_message_state` records `last_seen` only (`last_message_id` legacy/unused). `:cleardb` clears `user_message_state`. Postgres/MySQL CI smoke and WebSocket integration test cover visible replay SQL and second-connect wire replay.
- **Dependencies**: **github.com/jackc/pgx/v5** v5.10.0, **golang.org/x/crypto** v0.53.0, **golang.org/x/term** v0.44.0, **modernc.org/sqlite** v1.52.0 (SQLite 3.53.2).

## v1.2.0

**Released 2026-06-06.** Since **[v1.1.0](https://github.com/Cod-e-Codes/marchat/releases/tag/v1.1.0)**; compare [`v1.1.0...v1.2.0`](https://github.com/Cod-e-Codes/marchat/compare/v1.1.0...v1.2.0). Commits: **`git log v1.1.0..v1.2.0 --oneline`**.

- **Server**: WebSocket **Origin** checks compare parsed hostnames (no substring matching); optional **`MARCHAT_ALLOWED_ORIGINS`** allowlist. **`getClientIP`** and web-admin login rate limiting honor **`X-Forwarded-For`** / **`X-Real-IP`** only when the immediate peer is in **`MARCHAT_TRUSTED_PROXIES`** (comma-separated IPs or CIDRs).
- **Client**: Direct messages use the same E2E wire path as channel `text` when encryption is enabled (`encrypted` plus base64 nonce || ciphertext with the global key). Applies to **`:dm <user> <msg>`**, DM mode compose, and code snippets sent while a DM thread is open (**`:code`** / Alt+C). **Fix:** code snippets in DM mode route through the DM send path, not channel `text`.
- **Plugins**: Plugin store downloads validate SHA-256 checksums before extraction (HTTP and `file://`), reject oversize archives, parse `file://` paths correctly on Linux and Windows (registry and download URLs via **`plugin/fileurl`**), detect archive type from the URL path (including query strings), extract to a staging directory with zip-slip checks, roll back failed updates, set the execute bit on the plugin binary by exact name match after ZIP/TAR extract, and do not leave an empty plugin directory when install download fails.
- **Toolchain / dependencies**: Go **1.25.11** in **go.mod**, nested plugin modules, CI, and **Dockerfile** (stdlib fixes for **GO-2026-5037**, **GO-2026-5038**, **GO-2026-5039** reported by **govulncheck** on **1.25.10**); **golang.org/x/crypto** v0.52.0; **modernc.org/sqlite** v1.51.0 (was v1.50.0). Transitive **filippo.io/edwards25519** v1.2.0 (MySQL driver).
- **Packaging**: Version strings and URLs for **v1.2.0** in **install.ps1**, **install.sh**, **build-release.ps1**, **scripts/build-*.ps1/sh**, **README**, **SECURITY.md**, **.github/workflows/release.yml**, and **packaging/** (Homebrew, Scoop, winget **1.2.0** manifest set, Chocolatey, AUR) with **SHA256** from published release zips (**PACKAGING.md**, **packaging/ci/render-release-manifests.sh**).

## v1.1.0

**Released 2026-05-12.** Since **[v1.0.0](https://github.com/Cod-e-Codes/marchat/releases/tag/v1.0.0)**; compare [`v1.0.0...v1.1.0`](https://github.com/Cod-e-Codes/marchat/compare/v1.0.0...v1.1.0). Commits: **`git log v1.0.0..v1.1.0 --oneline`**.

- **Server**: **`messages.channel`** column (default **`general`**) with startup migration; channel messages persist and replay on the correct channel. Direct messages store **`recipient`**; reconnect history includes DM rows only for sender and recipient. **`TypingMessage`** with non-empty **`recipient`** uses the same DM delivery path as chat DMs.
- **Client**: Transcript and typing scoped to the active channel; DM thread sidebar (unread, hide, reappear), **`dm_state.json`** under the client config directory, footer shows the active DM peer, commands **`:dm`** / **`:dm off`** / **`:dms`** / **`:dmhide`**. **Fix:** **`:dmhide`** and **`:dms`** handled before **`:dm`** (prefix collision). **Typing:** DM compose sends optional **`recipient`** on the wire; reference TUI hides DM-scoped typing unless that thread is open and hides channel typing while a DM thread is open.
- **Diagnostics**: Client **`-doctor`** reports **`dm_state.json`** and E2E key source; server **`-doctor`** includes a DM history note.
- **Docs / protocol**: **ARCHITECTURE**, **PROTOCOL**, **README**, **TESTING**, **CONTRIBUTING**, **QUICKSTART**, **PLUGIN_ECOSYSTEM**, **docs/README** for DMs, typing **`recipient`**, and doctor output; optional graphical clients and **marchat-plugins** discovery where relevant.
- **Toolchain / dependencies**: Go **1.25.10** in **go.mod**, CI, **Dockerfile**; **golang.org/x/crypto** v0.51.0, **golang.org/x/term** v0.43.0; **github.com/jackc/pgx/v5** v5.9.2, **modernc.org/sqlite** v1.50.0, **github.com/go-sql-driver/mysql** v1.10.0.
- **CI**: Downstream **AUR** publish clones **aur.archlinux.org** over HTTPS before SSH push.
- **Packaging**: Version strings and URLs for **v1.1.0** in **install.ps1**, **install.sh**, **build-release.ps1**, **scripts/build-*.ps1/sh**, **README**, **SECURITY.md**, **.github/workflows/release.yml**, and **packaging/** (Homebrew, Scoop, winget **1.1.0** manifest set, Chocolatey, AUR). **SHA256** fields are **placeholders** (`000000...`) until replaced from published release zips (**PACKAGING.md**, **packaging/ci/render-release-manifests.sh**). Regenerate **packaging/aur/.SRCINFO** on Arch after final **PKGBUILD** checksums (**`makepkg --printsrcinfo`**).

## v1.0.0

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
