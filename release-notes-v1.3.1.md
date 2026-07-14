## v1.3.1

*Released: 14 July 2026*

### Server

- **Handshake replay**: **Fix** history replay sets `type: "text"` on channel messages so reconnect scrollback renders in the client (was omitted due to zero-value `omitempty`) (`server/handlers.go`)

### Toolchain and dependencies

- **Go**: **1.25.12** in **go.mod**, nested plugin modules, CI, and **Dockerfile** (stdlib fixes for reachable **GO-2026-5856** / **crypto/tls** ECH privacy leak and package-level **GO-2026-4970** / **os** symlink escape reported by **govulncheck** on **1.25.11**)
- **Dependencies**: **charm.land/bubbles/v2** v2.1.1 (textarea prompt styling fix); **golang.org/x/crypto** v0.54.0; **golang.org/x/term** v0.45.0 (transitive **golang.org/x/sys** v0.47.0, **golang.org/x/text** v0.40.0, **github.com/sahilm/fuzzy** v0.1.3)

### Version and packaging

- **Install and build defaults**: **install.ps1**, **install.sh**, **build-release.ps1**, **`scripts/build-windows.ps1`**, and **`scripts/build-linux.sh`** download and build against **v1.3.1** release assets on GitHub.
- **Docs and metadata**: **SECURITY.md** and **README** (version badge, install snippets, Docker tag) reference **v1.3.1**; canonical Homebrew, Scoop, winget, Chocolatey, and AUR templates in **`packaging/`** target **v1.3.1** (refresh zip SHA256 from published assets before `choco pack` / manifest validation).
- **Post-release helper**: **`scripts/post-release-v1.3.1.ps1`** for Chocolatey checksum sync and manifest render after assets upload.

### Assets

- marchat-v1.3.1-linux-amd64.zip
- marchat-v1.3.1-linux-arm64.zip
- marchat-v1.3.1-windows-amd64.zip
- marchat-v1.3.1-darwin-amd64.zip
- marchat-v1.3.1-darwin-arm64.zip

**Full Changelog:** https://github.com/Cod-e-Codes/marchat/compare/v1.3.0...v1.3.1

### Breaking changes

- **WebSocket JSON protocol**: No intentional breaking change; keystore and E2E wire encoding unchanged.

### Migration guide

- **Binaries**: use **v1.3.1** archives from this release page, or **install.ps1** / **install.sh** with their default version.
- **Client / server**: restart after upgrade; no database schema changes in this release.
- **Operators on v1.3.0**: upgrade server and client together if reconnect scrollback appeared blank after login (history replay `type` fix).
- **Packaging maintainers**: run **`scripts/post-release-v1.3.1.ps1`** or **`packaging/ci/render-release-manifests.sh`** after zips publish to refresh SHA256 in **`packaging/`** and downstream manifests.
