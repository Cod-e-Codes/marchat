# Package manager distribution

This repo ships **templates and checksums** under `packaging/` so you can publish marchat through Homebrew, winget, Scoop, Chocolatey, and the AUR without guessing URLs or archive layouts. Release zips always contain two binaries (names include the platform suffix), for example `marchat-client-linux-amd64` and `marchat-server-linux-amd64`.

## Published installs (current)

| Ecosystem | Status | User commands |
|-----------|--------|----------------|
| **Homebrew** (macOS / Linux) | Live tap [Cod-e-Codes/homebrew-marchat](https://github.com/Cod-e-Codes/homebrew-marchat) | `brew tap cod-e-codes/marchat` then `brew install marchat` |
| **Scoop** (Windows) | Live bucket [Cod-e-Codes/scoop-marchat](https://github.com/Cod-e-Codes/scoop-marchat) | `scoop bucket add marchat https://github.com/Cod-e-Codes/scoop-marchat` then `scoop install marchat` |
| **winget** (Windows) | Submission [microsoft/winget-pkgs#358094](https://github.com/microsoft/winget-pkgs/pull/358094) (pending review; after merge use `winget install Cod-e-Codes.Marchat`) | First-time contributors must reply to the PR with `@microsoft-github-policy-service agree` so the Microsoft CLA check passes |
| **Chocolatey** | Templates only in `packaging/chocolatey/` | Build and push per Chocolatey docs |
| **AUR** | Template only in `packaging/aur/` | Publish `marchat-bin` (or similar) under your AUR account |

Tap and bucket repos are maintained next to this project (for example under `packaging-forks/` on your machine). After each marchat release, update **both** the templates here and those repos, plus open a new winget-pkgs PR when the manifest version changes.

**Version discipline:** When you cut a release, bump the canonical version everywhere listed in `.cursor/rules/marchat.mdc` (install scripts, `build-release.ps1`, README badges and install snippets, workflow defaults, and the files under `packaging/`). Then recompute SHA256 for each zip (same names as in `.github/workflows/release.yml`).

**Checksums after a new tag:**

1. Download each `marchat-<tag>-<platform>.zip` from the GitHub release.
2. Replace `sha256` / `InstallerSha256` / `hash` fields in the packaging files for that tag.
3. Run validators when available (`brew audit`, `winget validate`, `choco pack` dry run).

Paths below are relative to the repo root.

## Homebrew (macOS and Linux)

**Published tap:** [github.com/Cod-e-Codes/homebrew-marchat](https://github.com/Cod-e-Codes/homebrew-marchat) (`Formula/marchat.rb`). Copy edits from `packaging/homebrew/marchat.rb` when you bump versions.

**Source file in this repo:** `packaging/homebrew/marchat.rb`

**core vs tap:** Submitting to `Homebrew/homebrew-core` is possible but review is stricter and slower; a tap is usually enough for pre-release tags and fast iteration.

## winget (Windows)

**Upstream:** [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs). Use your fork (for example [Cod-e-Codes/winget-pkgs](https://github.com/Cod-e-Codes/winget-pkgs)), add a version folder under `manifests/c/Cod-e-Codes/Marchat/<PackageVersion>/`, run `winget validate`, open a PR, and complete the CLA comment on the PR if the bot asks.

**Source layout in this repo (copy into your fork):** `packaging/winget/manifests/c/Cod-e-Codes/Marchat/0.11.0-beta.5/`

**Package identifier:** `Cod-e-Codes.Marchat` (publisher and app name; adjust only if you rename the publisher in manifests consistently).

**Notes:** Only the Windows amd64 zip is represented. Install is `zip` plus `NestedInstallerType: portable` so both `marchat-client` and `marchat-server` aliases are registered.

## Scoop (Windows)

**Published bucket:** [github.com/Cod-e-Codes/scoop-marchat](https://github.com/Cod-e-Codes/scoop-marchat) (`bucket/marchat.json`). Sync from `packaging/scoop/marchat.json` when you bump versions.

**Source file in this repo:** `packaging/scoop/marchat.json`

Users add the bucket, then `scoop install marchat` (name matches the manifest file name without `.json`). You can alternatively PR to [ScoopInstaller/Extras](https://github.com/ScoopInstaller/Extras) if maintainers accept it.

## Chocolatey (Windows)

**Typical approach:** A package folder with `marchat.nuspec` and `tools/chocolateyinstall.ps1`, built with `choco pack` and pushed to the Chocolatey community feed (or a private source).

**Source layout in this repo:** `packaging/chocolatey/`

Chocolatey `version` in the nuspec must follow their versioning rules; pre-releases often use a numeric prerelease label (see the nuspec comment).

## AUR (Arch Linux, binary package)

**Upstream:** Publish a `PKGBUILD` to the Arch User Repository (for example package name `marchat-bin`). Maintainers usually track the upstream repo separately from this tree.

**Source file in this repo:** `packaging/aur/PKGBUILD`

The PKGBUILD downloads the official **linux-amd64** or **linux-arm64** release zips (same static binaries as Termux arm64 users, documented in README).

## Install scripts and CI

Do **not** change the default install URL pattern in `install.sh` / `install.ps1` unless you intentionally switch download sources. Package managers are an extra path; GitHub release zips remain the source of truth and match `release.yml`.
