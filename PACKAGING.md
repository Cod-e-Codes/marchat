# Package manager distribution

The `packaging/` directory holds **templates** (and pinned checksums for the current release) for Homebrew, winget, Scoop, Chocolatey, and the AUR. Official release zips from GitHub each contain two binaries whose names include the platform suffix, for example `marchat-client-linux-amd64` and `marchat-server-linux-amd64`.

## Installing marchat

| Ecosystem | Notes | Command |
|-----------|-------|---------|
| **Homebrew** (macOS, Linux) | [Tap repo](https://github.com/Cod-e-Codes/homebrew-marchat) | `brew tap cod-e-codes/marchat` then `brew install marchat` |
| **Scoop** (Windows) | [Bucket repo](https://github.com/Cod-e-Codes/scoop-marchat) | `scoop bucket add marchat https://github.com/Cod-e-Codes/scoop-marchat` then `scoop install marchat` |
| **winget** (Windows) | Listed in [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs) after merge; track [PR #358094](https://github.com/microsoft/winget-pkgs/pull/358094) while pending | `winget install Cod-e-Codes.Marchat` |
| **Chocolatey** | Not published from this repo yet; see `packaging/chocolatey/` | N/A |
| **AUR** | Not published from this repo yet; see `packaging/aur/` | Use your helper or `yay`/`paru` once a package exists |

Other install paths (zip, `install.sh` / `install.ps1`, Docker) stay in [README.md](README.md).

## Files in this repo

| Path | Role |
|------|------|
| `packaging/homebrew/marchat.rb` | Homebrew formula template |
| `packaging/winget/manifests/...` | winget multi-file manifest set |
| `packaging/scoop/marchat.json` | Scoop manifest |
| `packaging/chocolatey/` | Chocolatey nuspec and install scripts |
| `packaging/aur/PKGBUILD` | AUR `-bin` style package |

## Release alignment

When you tag a release, bump the version everywhere listed in `.cursor/rules/marchat.mdc` (install scripts, `build-release.ps1`, README badges and snippets, workflow defaults, and every file under `packaging/` that embeds a version or URL).

After the release assets exist on GitHub:

1. Download each `marchat-<tag>-<platform>.zip` from the release.
2. Update `sha256` / `InstallerSha256` / `hash` fields in the packaging templates for that tag.
3. Run checks where applicable (`brew audit`, `winget validate`, `choco pack`).

Publish updates to **outbound** repos separately: push the tap `Formula/marchat.rb`, the Scoop `bucket/marchat.json`, a new folder under your **winget-pkgs** fork (then open a PR to Microsoft), Chocolatey push, or AUR Git as needed. Templates in **this** repo should match what you ship there.

## Homebrew

Published tap: [homebrew-marchat](https://github.com/Cod-e-Codes/homebrew-marchat). Formula source in marchat: `packaging/homebrew/marchat.rb`. Submitting to `Homebrew/homebrew-core` instead is possible but usually slower; a tap fits pre-releases and fast iteration.

## winget

Upstream is [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs). Maintainers use a fork, add `manifests/c/Cod-e-Codes/Marchat/<PackageVersion>/`, run `winget validate` on that folder, and open a PR. Package identifier: `Cod-e-Codes.Marchat`. The installer is a zip with `NestedInstallerType: portable` and two `PortableCommandAlias` entries for client and server. Microsoft may prompt first-time contributors to accept the CLA on the PR; follow the bot instructions there.

Example template path in this repo: `packaging/winget/manifests/c/Cod-e-Codes/Marchat/0.11.0-beta.5/` (duplicate the folder layout for new versions).

## Scoop

Published bucket: [scoop-marchat](https://github.com/Cod-e-Codes/scoop-marchat). Manifest source in marchat: `packaging/scoop/marchat.json`. Optional: propose the same manifest to [ScoopInstaller/Extras](https://github.com/ScoopInstaller/Extras) if their maintainers accept it.

## Chocolatey

Templates live in `packaging/chocolatey/`. Building: `choco pack` in that directory. Publishing to the community gallery needs a [Chocolatey account](https://community.chocolatey.org/) and API key; packages are moderated.

## AUR

Template: `packaging/aur/PKGBUILD` (binary package from official linux-amd64 and linux-arm64 zips). Publishing requires an [AUR account](https://aur.archlinux.org/), SSH key on the account, and a `.SRCINFO` generated with `makepkg --printsrcinfo` on Arch. Follow [AUR submission guidelines](https://wiki.archlinux.org/title/AUR_submission_guidelines). The canonical `PKGBUILD` for copying into the AUR Git repo is the one in this tree.

## Install scripts and CI

Do **not** change the default download URL pattern in `install.sh` or `install.ps1` unless you intentionally move binaries off GitHub releases. Package managers are additional channels; release zips stay the source of truth and match `.github/workflows/release.yml`.
