## v1.3.3

*Released: 3 August 2026*  
*Commit: 0e8a2e0*

### Server

- **File uploads**: **Fix** oversized file WebSocket messages use explicit read-limit handling with correct rejection logging; read limit accounts for base64 JSON wire size; rejects declared `size` and actual payload length above the limit; sends a System reply when the connection is still writable ([#114](https://github.com/Cod-e-Codes/marchat/issues/114))

### Client

- **File uploads**: **Fix** WebSocket close **1009** (message too big) shows a file-size error instead of a generic reconnect warning

### Plugins

- **Archive extraction**: **Fix** zip/tar extraction uses **`os.OpenRoot`** scoped writes and **`filepath.IsLocal`** entry validation (CodeQL **go/zipslip**)

### Dependencies

- **modernc.org/sqlite** v1.55.0 (SQLite 3.53.3; **modernc.org/libc** v1.74.1)
- **github.com/mattn/go-runewidth** v0.0.27

### Documentation

- **PROTOCOL** and **TESTING** document file rejection behavior; main-module coverage refreshed to **46.4%**

### Version and packaging

- **Install and build defaults**: **install.ps1**, **install.sh**, **build-release.ps1**, **`scripts/build-windows.ps1`**, and **`scripts/build-linux.sh`** target **v1.3.3** release assets on GitHub.
- **Docs and metadata**: **SECURITY.md** and **README** (version badge, install snippets, Docker tag) reference **v1.3.3**; canonical Homebrew, Scoop, winget, Chocolatey, and AUR templates in **`packaging/`** target **v1.3.3** (refresh zip SHA256 from published assets before `choco pack` / manifest validation).
- **Post-release helper**: **`scripts/post-release-v1.3.3.ps1`** for Chocolatey checksum sync and manifest render after assets upload.

### Assets

- marchat-v1.3.3-linux-amd64.zip
- marchat-v1.3.3-linux-arm64.zip
- marchat-v1.3.3-windows-amd64.zip
- marchat-v1.3.3-darwin-amd64.zip
- marchat-v1.3.3-darwin-arm64.zip

**Full Changelog:** https://github.com/Cod-e-Codes/marchat/compare/v1.3.2...v1.3.3

### Breaking changes

- **WebSocket JSON protocol**: No intentional breaking change; keystore and E2E wire encoding unchanged.

### Migration guide

- **Binaries**: use **v1.3.3** archives from this release page, or **install.ps1** / **install.sh** with their default version.
- **Client / server**: restart after upgrade; no database schema changes in this release.
- **Operators on v1.3.2**: upgrade server and client together if file uploads are used; plugin installs benefit from archive extraction hardening.
- **Packaging maintainers**: run **`scripts/post-release-v1.3.3.ps1`** or **`packaging/ci/render-release-manifests.sh`** after zips publish to refresh SHA256 in **`packaging/`** and downstream manifests.
