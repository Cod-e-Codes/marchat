# marchat Agent Skills

Project skills for Cursor Agent ([Agent Skills](https://cursor.com/docs/skills) format). Each skill is a folder with `SKILL.md`. Cursor discovers these automatically; invoke explicitly with `/skill-name` in Agent chat.

Tracked in git under `.cursor/rules/` and `.cursor/skills/`. Run `git add .cursor/rules/ .cursor/skills/` before commit so teammates get the same config on clone.

## Skills vs rules

| Layer | Location | Role |
|-------|----------|------|
| **Rules** | `.cursor/rules/marchat.mdc` | Always-on constraints, security, architecture facts |
| **Skills** | `.cursor/skills/*/` | Workflows: develop, test, debug, release, docs, domain areas |

When a task matches a skill description, read that skill's `SKILL.md` before acting.

For Cursor, dependencies, or platform behavior not defined in this repo, verify with web search or official documentation instead of assuming training-data defaults.

## Skill index

| Skill | Invoke | Scope |
|-------|--------|-------|
| [developing-marchat](developing-marchat/SKILL.md) | `/developing-marchat` | Go toolchain, modules, validation after code changes |
| [testing-marchat](testing-marchat/SKILL.md) | `/testing-marchat` | Tests, race, coverage, CI DB smoke, nested `plugin/sdk` |
| [debugging-marchat](debugging-marchat/SKILL.md) | `/debugging-marchat` | `-doctor`, env, WebSocket and connection issues |
| [releasing-marchat](releasing-marchat/SKILL.md) | `/releasing-marchat` | Version bumps, GitHub release, packaging, Docker |
| [writing-marchat-docs](writing-marchat-docs/SKILL.md) | `/writing-marchat-docs` | CHANGELOG, README, ARCHITECTURE, PROTOCOL, TESTING |
| [git-workflow-marchat](git-workflow-marchat/SKILL.md) | `/git-workflow-marchat` | Commit messages and PRs (no commit/push unless asked) |
| [database-marchat](database-marchat/SKILL.md) | `/database-marchat` | SQLite, Postgres, MySQL dialect and schema |
| [protocol-marchat](protocol-marchat/SKILL.md) | `/protocol-marchat` | Wire types, E2E encoding, protocol changes |
| [client-marchat](client-marchat/SKILL.md) | `/client-marchat` | Bubble Tea TUI, commands, keystore, reconnect |
| [server-marchat](server-marchat/SKILL.md) | `/server-marchat` | Hub, handlers, admin web, health, rate limits |
| [plugins-marchat](plugins-marchat/SKILL.md) | `/plugins-marchat` | SDK, host IPC, manager, licenses |

Nested skills under category folders are supported by Cursor but not required here; skill identity is the folder that contains `SKILL.md`.
