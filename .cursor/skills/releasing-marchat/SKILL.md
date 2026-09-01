---
name: releasing-marchat
description: >-
  Prepares marchat releases: version bumps, CHANGELOG, packaging checksums,
  GitHub Actions release workflow, and Docker tags. Use when tagging a release,
  bumping version strings, updating packaging manifests, or publishing assets.
---

# Releasing marchat

Publishing is triggered by a GitHub **release** (tag). Authoritative sources: `.github/workflows/release.yml`, `PACKAGING.md`, and the steps below. If workflow behavior is unclear, read those files or web search current GitHub Actions / `gh` CLI docs.

## Version bump checklist

Update the same version string everywhere:

- `install.ps1`, `install.sh`, `build-release.ps1`
- `scripts/build-windows.ps1`, `scripts/build-linux.sh`
- `README.md` (badge, install snippets, Docker tag pointer)
- `CHANGELOG.md` (new version section; move Unreleased items)
- `SECURITY.md`
- `.github/workflows/release.yml` (`workflow_dispatch` default / example)
- `packaging/` templates (Homebrew, Scoop, winget, Chocolatey, AUR)
- Go version sync if toolchain changes: `go.mod`, `Dockerfile`, `go.yml`, `release.yml`, `README.md`, `TESTING.md`

Details: `PACKAGING.md`, `.cursor/rules/marchat.mdc` Release Process section.

## After GitHub release assets exist

1. Download each `marchat-<tag>-<platform>.zip` from the release.
2. Refresh SHA256 in packaging templates (`packaging/ci/render-release-manifests.sh` with `RELEASE_TAG`).
3. Validate: `brew audit`, `winget validate`, `choco pack` where applicable.
4. Downstream publish uses secrets `PACKAGING_GITHUB_PAT`, `AUR_SSH_PRIVATE_KEY` (optional job `publish-downstream-packages`).

## Build notes

- Cross-builds: `CGO_ENABLED=0` for static binaries (workflow and Dockerfile).
- Archives: `.zip` for all platforms including Termux / linux-arm64.
- Docker: multi-arch `linux/amd64`, `linux/arm64` to Docker Hub on release.
- Asset upload: `gh release upload` in workflow (not third-party release actions on Node 20).

## GitHub release notes

`CHANGELOG.md` is the in-repo narrative. The published body lives on the GitHub release. Do **not** commit `release-notes*.md` (gitignored).

- Draft locally (for example `release-notes-vX.Y.Z.md`) and pass it to `gh release create --notes-file`. Leave the file untracked.
- After the prepare commit exists, set `*Commit: <short sha>*` to `git rev-parse --short` of the commit you tag.
- Do **not** include a `## Docker Image` section. The docker job appends Hub pull instructions with `gh release edit`. Putting that section in the notes file duplicates it when the image finishes.
- To edit a published body later, pass a real markdown file to `gh release edit --notes-file`. Do not rebuild notes in PowerShell strings (newlines get flattened).
- Chocolatey is not in `publish-downstream-packages`; after zips exist, `choco pack` in `packaging/chocolatey/` and `choco push` when asked. See `PACKAGING.md`.

## Changelog entry

Follow `writing-marchat-docs` skill and existing `CHANGELOG.md` sections: release date, compare links, grouped bullets (**Client**, **Server**, **Docs**, **Dependencies**, **Packaging**).

## Git

Do not commit or push unless the user asks. Provide a suggested commit message when done.
