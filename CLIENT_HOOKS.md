# Client external hooks (experimental)

**Status:** experimental. The hook protocol, environment variables, and payload fields may change or be removed in a future release without a major version bump. If you build integrations on this, pin a marchat version and watch release notes.

Client hooks run **local executables** you choose, fed one JSON object per event on **stdin** (newline-terminated). They are **not** server plugins: they only see what **your** client process sees, after decrypt on receive and before encrypt on send.

## Why this exists

- **Server plugins** automate hub-side behavior and see server-wide trust boundaries.
- **Built-in notifications** cover in-TUI alerts with a fixed feature set.
- **Client hooks** fill the gap: pipe chat events to **your** scripts, loggers, bridges, or external notification systems **without** new client dependencies or server changes.

Hooks are **side-effect only**: they cannot modify, block, or transform messages (runs are asynchronous).

## Security and trust

1. **Plaintext:** A receive hook runs after the client decrypts; a send hook runs with the plaintext you typed. Anyone who can read hook logs or the hook binary's behavior gets the same material as the TUI. This does **not** weaken wire encryption (the server still sees ciphertext for E2E traffic); it widens **local** exposure to whatever you execute.
2. **Absolute paths only:** `MARCHAT_CLIENT_HOOK_RECEIVE` and `MARCHAT_CLIENT_HOOK_SEND` must be absolute paths to a regular file. Relative paths are rejected to avoid surprising `PATH` / working-directory behavior.
3. **Trust the binary:** Hooks run as your user. Only point at programs you trust, same as running them manually.
4. **Timeouts:** Each invocation is killed if it exceeds the configured timeout (default 5s, max 120s) so a stuck script does not pile up forever.
5. **File messages:** Payloads include file **metadata** (`filename`, `size`) only; raw file bytes are never sent to hooks.

## Environment variables

| Variable | Meaning |
|----------|---------|
| `MARCHAT_CLIENT_HOOK_RECEIVE` | Absolute path to executable for inbound events (`message_received`). |
| `MARCHAT_CLIENT_HOOK_SEND` | Absolute path to executable for outbound composer sends (`message_send`). |
| `MARCHAT_CLIENT_HOOK_TIMEOUT_SEC` | Optional. Per-hook timeout in seconds (default `5`, max `120`). |
| `MARCHAT_CLIENT_HOOK_RECEIVE_TYPING` | Set to `1`, `true`, or `yes` to deliver **typing** indicators to the receive hook. Default: **off** (reduces log noise). |
| `MARCHAT_CLIENT_HOOK_DEBUG` | Set to `1`, `true`, or `yes` to log successful hook completion (duration) and label stdout as a debug preview. |
| `MARCHAT_HOOK_LOG` | Optional log file path for the **bundled** `example_hook` binary only. The marchat client does not read this variable. |

Unset hook paths mean that hook is disabled.

## Diagnostics (`-doctor`)

Run **`marchat-client -doctor`** (or **`-doctor-json`**) to inspect your client environment:

- The **Environment** section lists hook-related variables, including `MARCHAT_HOOK_LOG` for visibility, even though only `example_hook` uses it (with the same masking and truncation rules as other doctor output).
- If **`MARCHAT_CLIENT_HOOK_RECEIVE`** or **`MARCHAT_CLIENT_HOOK_SEND`** is set, doctor runs a **path check**: the value must be an absolute path to an existing regular file, matching what the client enforces at runtime.

Server doctor does **not** list client-only hook variables, even if they are set in the environment (the server never reads them; hiding them avoids noise when client and server are run from the same shell).

## Protocol

### Transport

- One UTF-8 JSON object per invocation, written to the process **stdin**, terminated with a single newline (`\n`).
- The client waits for the process to exit, up to the timeout. **Stdout** and **stderr** are captured; non-empty stdout is logged at INFO (see debug flag above). A non-zero exit or timeout is logged as a failure.

### Envelope

```json
{
  "event": "message_received | message_send",
  "version": 1,
  "message": { }
}
```

- **`version`:** Integer. Incremented when incompatible payload changes are introduced; until then, treat as `1`.
- **`event`:**
  - **`message_received`:** Fired for each inbound `shared.Message` on the chat WebSocket path after the client has applied decrypt when E2E is on. Typing is **excluded** unless `MARCHAT_CLIENT_HOOK_RECEIVE_TYPING` is enabled.
  - **`message_send`:** Fired for outbound text from the main composer path: global plaintext, global E2E (plaintext before encrypt), DMs, and `:` server/admin lines sent as `AdminCommandType`. Other UI actions (e.g. code snippet, file picker) may not invoke the send hook.

### `message` object

Mirrors [`shared.Message`](shared/types.go) fields where applicable, as a JSON object:

| Field | Notes |
|-------|--------|
| `type` | e.g. `text`, `dm`, `typing`, `reaction`, `admin_command`, … May be empty if the server omitted it on older history. |
| `sender`, `content`, `encrypted`, `message_id`, `recipient`, `edited`, `channel` | Same meaning as wire types. |
| `created_at` | RFC3339 nano UTC. **Omitted** when the client has no real timestamp (zero time), so you will not see `0001-01-01` sentinels. |
| `reaction` | Present for reaction events (`emoji`, `target_id`, `is_removal`). |
| `file` | For file messages: `{ "filename", "size" }` only; no `data`. |

Filter scripts on `message.type` / `event` as needed.

## Example: append-only logger

Build the bundled sample (from repo root). The directory is named **`_example_hook`** on purpose: Go omits paths whose first path element begins with **`_`** (or **`.`**) or is named **`testdata`** from **`./...`**, so this sample **`package main`** tree is not matched by **`go test ./...`** at the repository root. You still build or run it by passing that path explicitly, as below.

```bash
go build -o /tmp/marchat-hook-log ./client/exthook/_example_hook
```

Run the client with hooks (paths must be absolute on your OS):

```bash
export MARCHAT_CLIENT_HOOK_RECEIVE=/tmp/marchat-hook-log
export MARCHAT_CLIENT_HOOK_SEND=/tmp/marchat-hook-log
# Optional: override log path (otherwise uses $TMPDIR/marchat-client-hook.log)
export MARCHAT_HOOK_LOG=$HOME/marchat-hook.log
go run ./client
```

On Windows (PowerShell), use `Resolve-Path` for absolute paths and run `go run ./client` from the repository root (not `go run .`).

## Use cases (illustrative)

- Custom logging or archival to your own storage
- Webhooks or bridges to other chat systems
- Keyword-triggered local alerts beyond built-in notifications
- Development and integration testing

## Relationship to other systems

| Mechanism | Where it runs | Typical use |
|-----------|----------------|-------------|
| Server plugins | Server | Commands, hub automation, shared bots |
| Client hooks | Your machine | Personal automation, local pipelines |
| Second WebSocket client | Your machine / elsewhere | Full bots, independent sessions |

## Stability

Treat this document and the `version` field as the reference for experiments. For production-like integrations, prefer discussing on GitHub issues so breaking changes can be coordinated.
