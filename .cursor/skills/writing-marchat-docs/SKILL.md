---
name: writing-marchat-docs
description: >-
  Updates marchat documentation and changelogs to match code and release
  practice. Use when editing CHANGELOG.md, README.md, ARCHITECTURE.md,
  PROTOCOL.md, TESTING.md, or other project markdown.
paths:
  - "**/*.md"
---

# Writing marchat docs

Update docs when behavior, protocol, env vars, or coverage changes. Do not edit markdown the user did not ask for unless required by the same change set.

## Style

- Match existing voice: technical, direct, complete sentences.
- Use `**Bold label**:` for changelog section prefixes (**Client**, **Server**, **Docs**, **Fix:**).
- ASCII hyphen `-`, not Unicode em dash.
- No decorative emoji in docs or UI chrome descriptions (user message/reaction emoji in product behavior is fine to document).
- Link to `ARCHITECTURE.md`, `PROTOCOL.md`, `TESTING.md` instead of duplicating long specs.

## CHANGELOG.md

- **Unreleased** on `main` only until tag and publish.
- Each release: date, link to prior tag, `git log` hint, narrative bullets (not a raw commit dump).
- Call out breaking protocol or keystore changes explicitly.
- Dependency bumps: name and version.

## README.md

- Install paths, env vars, doctor, DB backends, proxy/WSS, coverage summary pointer to `TESTING.md`.
- Keep install script version snippets aligned with latest release when bumping version.

## ARCHITECTURE.md / PROTOCOL.md

- Normative for system design and wire JSON shapes.
- E2E: global symmetric ChaCha20-Poly1305; base64 nonce || ciphertext in `content` when `encrypted` is true.
- Do not describe chat E2E as X25519 key exchange.

## TESTING.md

- Regenerate coverage with `go test -coverprofile=...` and `go tool cover -func=...`.
- Document nested `plugin/sdk` separately from main module merge.
- Note doctor `osEnviron` / `environMu` parallel test constraint.

## Roadmap

Use `ROADMAP.md` for planned work. Do not document roadmap items as released unless they exist in code.

## Checklist

- [ ] Facts match current code (grep or read implementation)
- [ ] Version strings consistent across touched files
- [ ] Coverage numbers refreshed if tests changed materially
- [ ] No em dash introduced in new prose
