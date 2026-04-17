# Package manager distribution

The `packaging/` directory holds **templates** (and pinned checksums for the current release) for Homebrew, winget, Scoop, Chocolatey, and the AUR. Official release zips from GitHub each contain two binaries whose names include the platform suffix, for example `marchat-client-linux-amd64` and `marchat-server-linux-amd64`.

## Installing marchat

| Ecosystem | Notes | Command |
|-----------|-------|---------|
| **Homebrew** (macOS, Linux) | [Tap repo](https://github.com/Cod-e-Codes/homebrew-marchat) | `brew tap cod-e-codes/marchat` then `brew install marchat` |
| **Scoop** (Windows) | [Bucket repo](https://github.com/Cod-e-Codes/scoop-marchat) | `scoop bucket add marchat https://github.com/Cod-e-Codes/scoop-marchat` then `scoop install marchat` |
| **winget** (Windows) | Community manifests in [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs) ([initial package PR #358094](https://github.com/microsoft/winget-pkgs/pull/358094)); new versions ship via PR from your fork or the release workflow | `winget install Cod-e-Codes.Marchat` |
| **Chocolatey** | [community package](https://community.chocolatey.org/packages/marchat); templates in `packaging/chocolatey/`; prereleases use automated review | `choco install marchat` (when listed) |
| **AUR** (Arch) | [marchat-bin](https://aur.archlinux.org/packages/marchat-bin) on the AUR | `yay -S marchat-bin` or `paru -S marchat-bin` (or any AUR helper) |

Other install paths (zip, `install.sh` / `install.ps1`, Docker) stay in [README.md](README.md).

## Files in this repo

| Path | Role |
|------|------|
| `packaging/homebrew/marchat.rb` | Homebrew formula template |
| `packaging/winget/manifests/...` | winget multi-file manifest set |
| `packaging/scoop/marchat.json` | Scoop manifest |
| `packaging/chocolatey/` | Chocolatey nuspec and install scripts |
| `packaging/aur/PKGBUILD` | Canonical AUR package definition (edit here first) |
| `packaging/aur/.SRCINFO` | AUR metadata; regenerate on Arch with `makepkg --printsrcinfo > .SRCINFO` whenever `PKGBUILD` changes |

## Release alignment

When you tag a release, bump the version everywhere listed in install scripts, build-release.ps1, README badges and snippets, workflow defaults, and every file under packaging/ that embeds a version or URL.

After the release assets exist on GitHub:

1. Download each `marchat-<tag>-<platform>.zip` from the release.
2. Update `sha256` / `InstallerSha256` / `hash` fields in the packaging templates for that tag.
3. Run checks where applicable (`brew audit`, `winget validate`, `choco pack`).

Publish updates to **outbound** repos separately: push the tap `Formula/marchat.rb`, the Scoop `bucket/marchat.json`, a new folder under your **winget-pkgs** fork (then open a PR to Microsoft), Chocolatey push, or AUR Git as needed. Templates in **this** repo should match what you ship there.

### Automated publishing (GitHub release workflow)

After release assets are uploaded, the `publish-downstream-packages` job in [`.github/workflows/release.yml`](.github/workflows/release.yml) can refresh manifests from the published zips (via [`packaging/ci/render-release-manifests.sh`](packaging/ci/render-release-manifests.sh)) and push or PR them. Each destination is skipped if its secret is missing.

| Secret | Purpose |
|--------|---------|
| `PACKAGING_GITHUB_PAT` | Fine-grained or classic PAT with **contents** (and **pull requests** if you use fine-grained) on your forks: `{owner}/homebrew-marchat`, `{owner}/scoop-marchat`, `{owner}/winget-pkgs`. Used to push the tap and bucket, push a branch on the winget fork, and open a PR to `microsoft/winget-pkgs`. |
| `AUR_SSH_PRIVATE_KEY` | Private key whose public half is registered on [aur.archlinux.org](https://aur.archlinux.org/) for your maintainer account. Used to `git push` `PKGBUILD` and `.SRCINFO` to `ssh://aur@aur.archlinux.org/marchat-bin.git`. Multiline secret; include the full `BEGIN`/`END` lines. |

Fork names are derived from [`github.repository_owner`](https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/accessing-contextual-information-about-workflow-runs#github_context): `homebrew-marchat`, `scoop-marchat`, and `winget-pkgs` under the same owner as this repo. Chocolatey is not automated here (API key and moderation are separate).

**If `publish-downstream-packages` fails with `403` when pushing:** The workflow cannot use the default `GITHUB_TOKEN` to push to other repos (even under the same account). **`PACKAGING_GITHUB_PAT`** must be a personal access token that can write to those repositories. **Fine-grained:** grant **Contents: Read and write** on each of `homebrew-marchat`, `scoop-marchat`, and `winget-pkgs`, plus **Pull requests: Read and write** on `winget-pkgs` so the job can open a PR to `microsoft/winget-pkgs`. **Classic PAT:** **`repo`** scope is enough for those pushes. Re-run the failed workflow after updating the secret, or publish manifests manually from `packaging/` (see below).

## Homebrew

Published tap: [homebrew-marchat](https://github.com/Cod-e-Codes/homebrew-marchat). Formula source in marchat: `packaging/homebrew/marchat.rb`. Submitting to `Homebrew/homebrew-core` instead is possible but usually slower; a tap fits pre-releases and fast iteration.

## winget

Upstream is [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs). Maintainers use a fork, add `manifests/c/Cod-e-Codes/Marchat/<PackageVersion>/`, run `winget validate` on that folder, and open a PR. Package identifier: `Cod-e-Codes.Marchat`. The installer is a zip with `NestedInstallerType: portable` and two `PortableCommandAlias` entries for client and server. Microsoft may prompt first-time contributors to accept the CLA on the PR; follow the bot instructions there.

Example template path in this repo: `packaging/winget/manifests/c/Cod-e-Codes/Marchat/1.0.0/` (duplicate the folder layout for new versions).

**Checksums vs GitHub release timing:** Portable zip manifests embed SHA256. After you publish a release and the five platform zips exist on GitHub, run [`packaging/ci/render-release-manifests.sh`](packaging/ci/render-release-manifests.sh) with `RELEASE_TAG` set and copy `packaging-out/` into `packaging/` (and into your tap, bucket, winget fork, or AUR clone as needed) so local `winget validate`, `brew audit`, and `choco pack` match real bytes. The committed templates may carry placeholder hashes until that sync step.

**Windows Defender / VirusTotal:** Go-built portable zips are occasionally flagged as generic ML detections. If WinGet manual validation or Chocolatey scanning reports a trojan-style detection on an official release asset, submit the file and URL to [Microsoft Security Intelligence](https://www.microsoft.com/en-us/wdsi/filesubmission) (Defender) and reply on the WinGet PR with the submission id and SHA256, as in community triage threads. Chocolatey may show low VirusTotal counts on prereleases; stable packages still benefit from a clear upstream URL and checksum alignment.

## Scoop

Published bucket: [scoop-marchat](https://github.com/Cod-e-Codes/scoop-marchat). Manifest source in marchat: `packaging/scoop/marchat.json`. Optional: propose the same manifest to [ScoopInstaller/Extras](https://github.com/ScoopInstaller/Extras) if their maintainers accept it.

## Chocolatey

Templates live in `packaging/chocolatey/`. Building: `choco pack` in that directory. Publishing to the community gallery needs a [Chocolatey account](https://community.chocolatey.org/) and API key; packages are moderated.

## AUR

**Package:** [marchat-bin](https://aur.archlinux.org/packages/marchat-bin) (`marchat-client`, `marchat-server` from official linux-amd64 / linux-arm64 release zips).

**In this repo:** Maintain `packaging/aur/PKGBUILD` and `packaging/aur/.SRCINFO`. After any `PKGBUILD` edit, regenerate `.SRCINFO` on an Arch system (`makepkg --printsrcinfo > .SRCINFO` in that directory).

**Publishing updates:** Push `PKGBUILD` and `.SRCINFO` to the AUR with Git over SSH ([guidelines](https://wiki.archlinux.org/title/AUR_submission_guidelines#Submitting_packages)). Your account needs an SSH key on [aur.archlinux.org](https://aur.archlinux.org/).

**Verify build (Docker on Windows):** `docker run --rm archlinux:latest bash -lc "pacman -Sy --noconfirm --needed base-devel git && useradd -m build && su build -c 'cd /home/build && git clone https://aur.archlinux.org/marchat-bin.git && cd marchat-bin && makepkg -f --noconfirm'"`

## Install scripts and CI

Do **not** change the default download URL pattern in `install.sh` or `install.ps1` unless you intentionally move binaries off GitHub releases. Package managers are additional channels; release zips stay the source of truth and match `.github/workflows/release.yml`.
