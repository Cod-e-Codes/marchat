## v1.3.4

*Released: 3 August 2026*  
*Commit: TBD*

### Server

- **File uploads**: **Fix** oversized uploads under the WebSocket DoS read ceiling (**32 MiB**) get a System reply on a live connection; `SetReadLimit` sits above policy wire size so gorilla does not close with empty **1009** before the reply can flush. Residual over-ceiling reads log `ErrReadLimit` only ([#114](https://github.com/Cod-e-Codes/marchat/issues/114) follow-up)

### Documentation

- **PROTOCOL**, **TESTING**, and **README** document DoS ceiling vs policy reject; main-module coverage refreshed to **46.3%**

### Version and packaging

- **Install and build defaults**: **install.ps1**, **install.sh**, **build-release.ps1**, **`scripts/build-windows.ps1`**, and **`scripts/build-linux.sh`** target **v1.3.4** release assets on GitHub.
- **Docs and metadata**: **SECURITY.md** and **README** (version badge, install snippets, Docker tag) reference **v1.3.4**; canonical Homebrew, Scoop, winget, Chocolatey, and AUR templates in **`packaging/`** target **v1.3.4** (refresh zip SHA256 from published assets before `choco pack` / manifest validation).
- **Post-release helper**: **`scripts/post-release-v1.3.4.ps1`** for Chocolatey checksum sync and manifest render after assets upload.

### Assets

- marchat-v1.3.4-linux-amd64.zip
- marchat-v1.3.4-linux-arm64.zip
- marchat-v1.3.4-windows-amd64.zip
- marchat-v1.3.4-darwin-amd64.zip
- marchat-v1.3.4-darwin-arm64.zip

**Full Changelog:** https://github.com/Cod-e-Codes/marchat/compare/v1.3.3...v1.3.4

### Breaking changes

- **WebSocket JSON protocol**: No intentional breaking change; keystore and E2E wire encoding unchanged.

### Migration guide

- **Binaries**: use **v1.3.4** archives from this release page, or **install.ps1** / **install.sh** with their default version.
- **Client / server**: restart after upgrade; no database schema changes in this release.
- **Operators on v1.3.3**: upgrade the server for the client-facing oversized-file System message; marchat clients already map close **1009** from v1.3.3.
- **Packaging maintainers**: run **`scripts/post-release-v1.3.4.ps1`** or **`packaging/ci/render-release-manifests.sh`** after zips publish to refresh SHA256 in **`packaging/`** and downstream manifests.

## Docker Image

A multi-architecture Docker image (linux/amd64, linux/arm64) is available on Docker Hub:

```bash
docker pull codecodesxyz/marchat:v1.3.4
# or use latest tag
docker pull codecodesxyz/marchat:latest
```
