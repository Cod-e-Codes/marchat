---
name: git-workflow-marchat
description: >-
  Drafts conventional commit messages and pull request bodies for marchat
  without committing or pushing unless explicitly requested. Use when the user
  asks for a commit message, PR description, or git workflow help; also read at
  the end of every substantive implementation task.
---

# Git workflow for marchat

## Default behavior

- **Do not** run `git commit` or `git push` unless the user explicitly asks.
- **Always** provide a suggested commit message when implementation work is complete, even if the user did not ask to commit.
- Scope the message to **all** uncommitted changes (`git status`, `git diff`), including prior uncommitted work from earlier turns in the same session.
- Never amend, force-push, or skip hooks unless the user explicitly requests and conditions are met.

## Commit message format

Conventional commits: `feat:`, `fix:`, `chore:`, `docs:`, `style:`, `ci:`, `test:`.

```
<type>(<optional scope>): <short summary>

<optional body: why, not a file list>
```

Examples:

```
fix(server): use dialect boolean literals in pin toggle SQL

Postgres rejected integer comparison on boolean columns.
```

```
docs: refresh TESTING coverage from merged profile
```

## When user asks to commit

1. Parallel: `git status`, `git diff`, `git log -1 --oneline` (style reference).
2. Stage only relevant files; warn on secrets (`.env`, keys).
3. Commit via HEREDOC message; verify with `git status`.
4. If pre-commit hook fails, fix and **new** commit (no amend unless rules allow).

## Pull requests

Use `gh` when user asks to create a PR:

1. Parallel: status, diff, tracking branch, `git log main...HEAD`, `git diff main...HEAD`.
2. Push with `-u` only if needed and user asked.
3. `gh pr create` with Summary and Test plan sections.

## Breaking changes

Note in commit body and `CHANGELOG.md` when protocol or keystore format breaks compatibility.
