# marchat quickstart

Get a **server** and **client** running in a few minutes. For full detail, see [README.md](README.md).

## What you need

- **Server**: `MARCHAT_ADMIN_KEY` (secret) and `MARCHAT_USERS` (comma-separated admin usernames). No other env vars are required for a local trial.
- **Client**: A username and the WebSocket URL (default path is `/ws`).
- **Optional**: [Go 1.25.9+](https://go.dev/dl/) only if you build from source; `openssl` (or any way to produce 64 hex chars) to generate the admin key.

## 1. Install binaries

Pick one:

- **GitHub releases**: Download the zip for your OS/arch from [releases](https://github.com/Cod-e-Codes/marchat/releases), unpack, and use `marchat-server` and `marchat-client`. **Termux (aarch64):** use **`linux-arm64`** (same static `GOOS=linux` build as Linux ARM64 servers).
- **Windows (PowerShell)**:

  ```powershell
  iwr -useb https://raw.githubusercontent.com/Cod-e-Codes/marchat/main/install.ps1 | iex
  ```

- **Docker**: `docker pull codecodesxyz/marchat` (pin a tag in production; see [README.md](README.md#installation)).
- **Homebrew** ([tap](https://github.com/Cod-e-Codes/homebrew-marchat)): `brew tap cod-e-codes/marchat` then `brew install marchat`.
- **Scoop** ([bucket](https://github.com/Cod-e-Codes/scoop-marchat)): `scoop bucket add marchat https://github.com/Cod-e-Codes/scoop-marchat` then `scoop install marchat`.
- **winget**: `winget install Cod-e-Codes.Marchat`. Maintainer workflow and checksums: [PACKAGING.md](PACKAGING.md#installing-marchat).

## 2. Create secrets

```bash
openssl rand -hex 32
```

Use the output as `MARCHAT_ADMIN_KEY`. Choose admin usernames (e.g. `alice,bob`) for `MARCHAT_USERS`.

## 3. Run the server

### Environment variables (simplest)

```bash
export MARCHAT_ADMIN_KEY="<paste-hex-key>"
export MARCHAT_USERS="alice,bob"
./marchat-server
```

Default listen port is **8080** (all interfaces, `:8080`). Optional: `--admin-panel` or `--web-panel` for admin UIs (see [README.md](README.md)).

**`--interactive`:** Runs the first-time setup wizard **only** if `MARCHAT_ADMIN_KEY` or `MARCHAT_USERS` is missing (unset in the environment and not supplied in `config/.env`). If both are already configured, this flag has no extra effect: the server starts like `./marchat-server`. Without required config and without `--interactive`, the server exits with an error pointing you to set env vars or use `--interactive`.

### Config file (repo / long-running)

From the repo (or any layout where the server’s config dir is `config/`):

```bash
cp env.example config/.env
# Edit config/.env: set MARCHAT_ADMIN_KEY and MARCHAT_USERS
./marchat-server
```

The server reads `config/.env` with **overload** semantics: keys present in the file override the same variables already in the process environment. Restart after edits. See [README.md](README.md#server-config-env-vs-process-environment).

### Docker (one container)

```bash
docker run -d -p 8080:8080 \
  -e MARCHAT_ADMIN_KEY="$(openssl rand -hex 32)" \
  -e MARCHAT_USERS=alice,bob \
  codecodesxyz/marchat
```

### Docker Compose

The sample [docker-compose.yml](docker-compose.yml) exposes the port and database volume; you **must** add `MARCHAT_ADMIN_KEY` and `MARCHAT_USERS` (for example via a gitignored `.env` next to the compose file). See [README.md](README.md#docker-compose-local-development).

## 4. Connect the client

Default URL: `ws://localhost:8080/ws`.

**Admin (can run admin commands after auth):**

```bash
./marchat-client --username alice --admin --admin-key "<same-as-MARCHAT_ADMIN_KEY>" --server ws://localhost:8080/ws
```

**Regular user:**

```bash
./marchat-client --username user1 --server ws://localhost:8080/ws
```

Or run `./marchat-client` with no flags and use the interactive config flow.

For optional graphical clients, see [README.md](README.md#optional-graphical-clients).

**TLS / `wss://`:** Set `MARCHAT_TLS_CERT_FILE` and `MARCHAT_TLS_KEY_FILE` on the server, or put Caddy (or another proxy) in front. Local Caddy + helper scripts: [deploy/CADDY-REVERSE-PROXY.md](deploy/CADDY-REVERSE-PROXY.md).

## 5. Build from source (optional)

```bash
git clone https://github.com/Cod-e-Codes/marchat.git && cd marchat
go mod tidy
go build -o marchat-server ./cmd/server
go build -o marchat-client ./client
```

Linux clipboard for the client: install `xclip` (or your distro’s equivalent). See [README.md](README.md#from-source).

## 6. Verify and troubleshoot

```bash
./marchat-server -doctor
./marchat-client -doctor
```

Use `-doctor-json` for machine-readable output. To skip the GitHub “latest release” check: `MARCHAT_DOCTOR_NO_NETWORK=1`. See [README.md](README.md#diagnostics--doctor) and [ARCHITECTURE.md](ARCHITECTURE.md).

## 7. First minutes in the TUI

- **`Ctrl+H`**: help overlay.
- **Channels**: You start in `#general`; use `:join <name>`, `:leave`, `:channels`.
- **DMs**: `:dm` (see [README.md](README.md#user-commands)).
- **Quit**: `:q` or your terminal’s usual exit.

## Where to read next

| Topic | Doc |
|--------|-----|
| Doc index | [docs/README.md](docs/README.md) |
| Release history | [CHANGELOG.md](CHANGELOG.md) |
| All options, commands, hotkeys | [README.md](README.md) |
| Components, config paths, doctor | [ARCHITECTURE.md](ARCHITECTURE.md) |
| WebSocket message shapes | [PROTOCOL.md](PROTOCOL.md) |
| Threat model, reporting | [SECURITY.md](SECURITY.md) |
| Themes | [THEMES.md](THEMES.md) |
| Desktop / bell / quiet hours | [NOTIFICATIONS.md](NOTIFICATIONS.md) |
| Plugins | [PLUGIN_ECOSYSTEM.md](PLUGIN_ECOSYSTEM.md), [plugin/README.md](plugin/README.md) |
| Tests and CI expectations | [TESTING.md](TESTING.md) |
| Package managers | [PACKAGING.md](PACKAGING.md) |
| Contributing | [CONTRIBUTING.md](CONTRIBUTING.md) |
