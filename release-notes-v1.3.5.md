## v1.3.5

*Released: 15 August 2026*

### Server

- **Moderation**: **Fix** `:kick` / `:ban` reject self-targets; kick is online-only (`ErrKickNotConnected`); ban remains offline-capable ([#115](https://github.com/Cod-e-Codes/marchat/issues/115), [#116](https://github.com/Cod-e-Codes/marchat/issues/116))
- **Messages**: **Fix** empty or whitespace-only plaintext on `text`, `dm`, and `edit` is rejected when `encrypted` is false ([#117](https://github.com/Cod-e-Codes/marchat/issues/117))
- **SQLite**: **Fix** `InitDB` applies `busy_timeout` / WAL via DSN on every connection and uses a single open connection so concurrent inserts no longer fail with `SQLITE_BUSY` ([#118](https://github.com/Cod-e-Codes/marchat/issues/118))
- **Persistence**: active permanent bans and unexpired temp kicks reload from `ban_history` on hub start (`expires_at` NULL = permanent). Permanent bans are presence-only in memory. `:ban` / `:kick` close any open row before insert and persist to the DB before in-memory state
- **Schema**: versioned `MigrateSchema` (`schema_version`) hard-fails when required tables or `ban_history.expires_at` are missing. SQLite/Postgres wrap each version in a transaction; MySQL DDL cannot, so `schema_version` is recorded after a successful apply
- **Lookups**: connected-user lookups use an O(1) `clientsByUsername` map

### Documentation

- **ARCHITECTURE** documents `MigrateSchema` / `schema_version` / `ban_history.expires_at`; **TESTING** lint pins match CI; main-module coverage **47.8%**

### Toolchain and CI

- **Go 1.25.13** in **go.mod**, nested plugin modules, CI, and **Dockerfile** (stdlib fixes for the six reachable findings on **1.25.12**)
- **CI** runs **`govulncheck ./...`** without **`-show verbose`**. Pins **golangci-lint** v2.12.2 and **govulncheck** v1.6.0; `.golangci.yml` (v2) enables govet/ineffassign/staticcheck with `all` minus `ST*`/`QF*`

### Dependencies

- **modernc.org/sqlite** v1.56.0; **github.com/lucasb-eyer/go-colorful** v1.4.1; **golang.org/x/crypto** v0.55.0; **github.com/charmbracelet/x/ansi** v0.11.8; **charm.land/lipgloss/v2** v2.0.6

### Version and packaging

- **Install and build defaults**: **install.ps1**, **install.sh**, **build-release.ps1**, **`scripts/build-windows.ps1`**, and **`scripts/build-linux.sh`** target **v1.3.5** release assets on GitHub.
- **Docs and metadata**: **SECURITY.md** and **README** (version badge, install snippets, Docker tag) reference **v1.3.5**; canonical Homebrew, Scoop, winget, Chocolatey, and AUR templates in **`packaging/`** target **v1.3.5** (refresh zip SHA256 from published assets before `choco pack` / manifest validation).
- **Post-release helper**: **`scripts/post-release-v1.3.5.ps1`** for Chocolatey checksum sync and manifest render after assets upload.

### Assets

- marchat-v1.3.5-linux-amd64.zip
- marchat-v1.3.5-linux-arm64.zip
- marchat-v1.3.5-windows-amd64.zip
- marchat-v1.3.5-darwin-amd64.zip
- marchat-v1.3.5-darwin-arm64.zip

**Full Changelog:** https://github.com/Cod-e-Codes/marchat/compare/v1.3.4...v1.3.5

### Breaking changes

- **WebSocket JSON protocol**: No intentional breaking change; keystore and E2E wire encoding unchanged.
- **Database**: Existing databases are migrated at server start (`schema_version`, `ban_history.expires_at`). Startup **hard-fails** if required tables or that column are missing after migration. Back up the DB before upgrading.

### Migration guide

- **Binaries**: use **v1.3.5** archives from this release page, or **install.ps1** / **install.sh** with their default version.
- **Client / server**: restart after upgrade. The server applies schema migrations on start; SQLite/Postgres wrap each version in a transaction; MySQL DDL cannot participate in a wrapping transaction.
- **Operators on v1.3.4**: upgrade the server for durable kick/ban across restart and the schema migrator. Existing open `ban_history` rows without `expires_at` load as permanent.
- **Packaging maintainers**: run **`scripts/post-release-v1.3.5.ps1`** or **`packaging/ci/render-release-manifests.sh`** after zips publish to refresh SHA256 in **`packaging/`** and downstream manifests.
