## v1.3.2

*Released: 15 July 2026*  
*Commit: d1efa76*

### Server

- **Sender stamping**: **Security** server overwrites client-supplied `sender` on text and file outbound paths (`stampSenderTimedOutbound` in `server/client.go`)
- **Content validation**: **Security** rejects NUL bytes in persistable content before insert (`server/message_validate.go`)
- **Persist failure**: **Security** does not broadcast when message persistence fails (`server/client.go`)

### Client

- **Desktop notifications**: **Security** Windows toast XML built in Go with `xml.EscapeText` and shown via `powershell -EncodedCommand`; macOS `osascript` uses `strconv.Quote` string literals (no shell interpolation of wire content) (`client/notification_desktop.go`)

### Documentation

- **Trust boundaries**: **PROTOCOL**, **SECURITY**, **ARCHITECTURE**, and **TESTING** document server sender stamping and safe desktop notification paths for untrusted wire content

### Version and packaging

- **Install and build defaults**: **install.ps1**, **install.sh**, **build-release.ps1**, **`scripts/build-windows.ps1`**, and **`scripts/build-linux.sh`** download and build against **v1.3.2** release assets on GitHub.
- **Docs and metadata**: **SECURITY.md** and **README** (version badge, install snippets, Docker tag) reference **v1.3.2**; canonical Homebrew, Scoop, winget, Chocolatey, and AUR templates in **`packaging/`** target **v1.3.2** (refresh zip SHA256 from published assets before `choco pack` / manifest validation).
- **Post-release helper**: **`scripts/post-release-v1.3.2.ps1`** for Chocolatey checksum sync and manifest render after assets upload.

### Assets

- marchat-v1.3.2-linux-amd64.zip
- marchat-v1.3.2-linux-arm64.zip
- marchat-v1.3.2-windows-amd64.zip
- marchat-v1.3.2-darwin-amd64.zip
- marchat-v1.3.2-darwin-arm64.zip

**Full Changelog:** https://github.com/Cod-e-Codes/marchat/compare/v1.3.1...v1.3.2

### Breaking changes

- **WebSocket JSON protocol**: No intentional breaking change; keystore and E2E wire encoding unchanged.

### Migration guide

- **Binaries**: use **v1.3.2** archives from this release page, or **install.ps1** / **install.sh** with their default version.
- **Client / server**: restart after upgrade; no database schema changes in this release.
- **Operators on v1.3.1**: upgrade the server if untrusted clients can connect (sender stamping fix); upgrade the client if desktop notifications are enabled (Alt+N).
- **Packaging maintainers**: run **`scripts/post-release-v1.3.2.ps1`** or **`packaging/ci/render-release-manifests.sh`** after zips publish to refresh SHA256 in **`packaging/`** and downstream manifests.
