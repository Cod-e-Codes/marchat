# Caddy reverse proxy + WSS for marchat

This document describes how to run **marchat-server** on the host (plain HTTP on port **8080**), put **Caddy** in Docker in front of it (TLS on host port **8443**), and connect the client with **`wss://`**. It also lists **source changes** in this repo that make that setup reliable.

Commands are given in two forms where it matters: **Unix shell** (`bash`) and **Windows PowerShell**.

## Architecture

```
marchat-client  --wss-->  localhost:8443  (Docker publishes host 8443 -> container 443)
                              |
                           Caddy (TLS terminate, WebSocket-aware reverse_proxy)
                              |
                    host.docker.internal:8080
                              |
                    marchat-server (host process, reads config/.env)
```

- **Do not** expose **8080** to the internet if you only intend to serve through Caddy; forward **8443** on the router instead.
- Caddy reaches the server via **`host.docker.internal`** (see `docker-compose.proxy.yml`). Docker Compose adds **`host-gateway`** on Linux, Windows, and macOS so this matches a typical dev setup.

---

## Prerequisites

- **Go 1.25.8+** (for building from source).
- **Docker** with **Compose v2** (Docker Engine on Linux; **Docker Desktop** on Windows or macOS is fine).
- **marchat** repo cloned; server config directory **`config/`** exists.

---

## Step 1: Server configuration (`config/.env`)

Create **`config/.env`** as **UTF-8 without a BOM** (a BOM breaks parsing and you will see errors like `unexpected character` near the first variable).

Minimum:

```env
MARCHAT_PORT=8080
MARCHAT_ADMIN_KEY=<64-char-hex-or-your-secret>
MARCHAT_USERS=admin1
```

Optional (global E2E, same value on every client that should read encrypted channels):

```env
MARCHAT_GLOBAL_E2E_KEY=<base64-32-bytes>
```

Generate examples:

- Admin key: any cryptographically strong secret (often 64 hex chars).
- Global E2E key: 32 random bytes in base64, e.g. **`openssl rand -base64 32`** (Unix) or WSL/Git Bash on Windows.

---

## Step 2: Build binaries (correct Go settings)

Release-style local build:

- **`CGO_ENABLED=0`** (static binary, same as Docker/release; uses pure-Go SQLite).
- **`-ldflags`** set **`shared.ClientVersion`**, **`shared.ServerVersion`**, **`shared.BuildTime`**, **`shared.GitCommit`**.

**Unix (Linux binary on Linux; set `GOOS` / `GOARCH` for cross-compile):**

```bash
chmod +x scripts/build-linux.sh scripts/connect-local-wss.sh   # once, if needed
./scripts/build-linux.sh
# macOS host targeting macOS:
#   GOOS=darwin GOARCH=arm64 ./scripts/build-linux.sh
# (Script name is historical; it honors GOOS/GOARCH.)
```

**Windows (amd64 `.exe` in repo root):**

```powershell
.\scripts\build-windows.ps1
```

Or mirror the same flags from **`build-release.ps1`** / **`.github/workflows/release.yml`**. The release workflow sets **`CGO_ENABLED=0`** on the **`build`** job (static cross-compiles, pure-Go SQLite) and uses a **`resolve-version`** job so the tag/input version is available to Docker and matrix builds (matrix jobs cannot publish job outputs).

---

## Step 3: Start marchat-server

From the **repository root** (so default config dir resolves to **`./config`** and loads **`config/.env`**):

**Unix:**

```bash
./marchat-server
```

**Windows:**

```powershell
.\marchat-server.exe
```

Confirm:

**Unix:**

```bash
curl -fsS http://127.0.0.1:8080/health/simple
# Expect: OK
```

**Windows:**

```powershell
Invoke-WebRequest http://127.0.0.1:8080/health/simple -UseBasicParsing
# Expect: OK
```

**Restart the server** after editing **`config/.env`**.

---

## Step 4: Caddy (Docker Compose)

### 4.1 Files

- **`docker-compose.proxy.yml`** – runs **`caddy:2-alpine`**, maps **`8443:443`**, mounts **`deploy/caddy/Caddyfile`**, loads **`deploy/caddy/proxy.env.example`** and merges optional **`deploy/caddy/proxy.env`** (gitignored), adds **`host.docker.internal:host-gateway`**.
- **`deploy/caddy/Caddyfile`** – TLS + `reverse_proxy` to **`host.docker.internal:8080`**; optional extra TLS names via **`MARCHAT_CADDY_EXTRA_HOSTS`** (see **`proxy.env.example`** / local **`proxy.env`**).
- **`deploy/caddy/proxy.env.example`** – tracked defaults (empty **`MARCHAT_CADDY_EXTRA_HOSTS`**). Copy to **`deploy/caddy/proxy.env`** (ignored by git) to set public IP or DNS names on the site line so **`tls internal`** certificates match how remote clients connect.

### 4.2 Caddyfile rules that matter

1. **Named site block, not bare `:443`**

   A site like **`:443 { ... }`** with **`tls internal`** caused TLS **alert internal_error** (clients saw **`remote error: tls: internal error`**). Use explicit names:

   ```caddyfile
   localhost, 127.0.0.1{$MARCHAT_CADDY_EXTRA_HOSTS} {
       tls internal
       reverse_proxy host.docker.internal:8080 {
           flush_interval -1
           header_up X-Forwarded-Proto https
       }
   }
   ```

   **`{$MARCHAT_CADDY_EXTRA_HOSTS}`** comes from Compose **`env_file`** (**`proxy.env.example`**, then optional **`proxy.env`**). For internet clients dialing your **public IP**, copy the example to **`deploy/caddy/proxy.env`** and set e.g. **`MARCHAT_CADDY_EXTRA_HOSTS=, 203.0.113.5`** (leading comma + space), then recreate Caddy (see below).

2. **`flush_interval -1`**

   Recommended for long-lived WebSocket streams so Caddy does not buffer responses in a way that stalls the connection.

3. **First-time or broken TLS state**

   If you previously used a bad `:443` config, **recreate Caddy’s volumes** once so internal certs are re-issued:

   **Unix:**

   ```bash
   docker compose -f docker-compose.proxy.yml down
   docker volume rm marchat_caddy_data marchat_caddy_config  # ignore errors if missing
   docker compose -f docker-compose.proxy.yml up -d
   ```

   **Windows (PowerShell):**

   ```powershell
   docker compose -f docker-compose.proxy.yml down
   docker volume rm marchat_caddy_data marchat_caddy_config   # ignore errors if missing
   docker compose -f docker-compose.proxy.yml up -d
   ```

### 4.3 Start Caddy

**Unix or Windows (same):**

```bash
docker compose -f docker-compose.proxy.yml up -d
```

---

## Step 5: Host firewall

Allow inbound **TCP 8443** so LAN or internet clients can reach Caddy.

**Windows (elevated PowerShell):**

```powershell
New-NetFirewallRule -DisplayName "marchat WSS (8443)" -Direction Inbound -LocalPort 8443 -Protocol TCP -Action Allow
```

**Linux (`ufw`):**

```bash
sudo ufw allow 8443/tcp
sudo ufw reload
```

**Linux (`firewalld`):**

```bash
sudo firewall-cmd --permanent --add-port=8443/tcp
sudo firewall-cmd --reload
```

Adjust if you use another firewall or bind Docker differently.

---

## Step 6: Client connection

### URL and TLS

- Prefer **`wss://localhost:8443/ws`** while using Caddy’s **`tls internal`** certificates.
- Until you use a real public CA certificate on Caddy, use **`--skip-tls-verify`** on the client.

The **Go client** in this repo forces **ALPN `http/1.1`** for `wss` and fixes **SNI** when the URL uses loopback IPs (see **Source changes** below).

### E2E

- Set **`MARCHAT_GLOBAL_E2E_KEY`** in the environment to match the server (or rely on a client-generated key only if you control all peers).
- **`--e2e`** and **`--keystore-passphrase`** unlock the local keystore; passphrase is **not** the admin key.

### Convenience scripts

**Unix:**

```bash
./scripts/connect-local-wss.sh
# Non-interactive keystore pass:
#   KEYSTORE_PASS='yourpass' ./scripts/connect-local-wss.sh
```

**Windows:**

```powershell
.\scripts\connect-local-wss.ps1 -KeystorePass yourpass
```

Both read **`config/.env`** for admin key and global E2E key, run the client against **`wss://localhost:8443/ws`** with **`--skip-tls-verify`**. Username is **`MARCHAT_CLIENT_USERNAME`** if set, otherwise the **first** name in **`MARCHAT_USERS`**.

**PowerShell scripts:** use **ASCII** punctuation in strings (avoid Unicode dashes) so Windows PowerShell 5.1 does not mangle encoding.

### Admin key mismatch (“invalid admin key”)

If the shell or IDE exports a **stale `MARCHAT_*`**, the server used to keep the old value because **`godotenv.Load`** does not override existing env vars. This repo now uses **`godotenv.Overload`** for **`config/.env`** so the **file wins** (see **`config/config.go`**). Restart the server after changing code.

---

## Step 7: Remote users (LAN / internet)

- Add every host or IP clients will use to **`MARCHAT_CADDY_EXTRA_HOSTS`** in local **`deploy/caddy/proxy.env`** (copy from **`proxy.env.example`** if needed; see comments there), then recreate Caddy so **TLS SANs** match **SNI**.
- **LAN:** clients use **`wss://<your-LAN-IP>:8443/ws`** (firewall must allow 8443).
- **Internet:** router **port forward TCP** external **8443** (or 443) to this PC **8443**; clients use your public IP or DNS name. Prefer a real TLS certificate on Caddy for production and **remove `--skip-tls-verify`**.
- Optional: fill **`deploy/REMOTE-INVITE.template.md`** for copy-paste guest instructions (do not commit secrets).

---

## Source code changes (summary)

| Area | File | Change |
|------|------|--------|
| **Deploy** | `deploy/caddy/Caddyfile` | **`localhost, 127.0.0.1{$MARCHAT_CADDY_EXTRA_HOSTS} { ... }`** instead of **`:443`**, **`tls internal`**, **`reverse_proxy host.docker.internal:8080`** with **`flush_interval -1`** and **`X-Forwarded-Proto`**. |
| **Deploy** | `deploy/caddy/proxy.env.example` | Tracked defaults for Caddy env; optional local **`deploy/caddy/proxy.env`** (gitignored) overrides **`MARCHAT_CADDY_EXTRA_HOSTS`** for public IP / DNS. |
| **Deploy** | `docker-compose.proxy.yml` | Caddy service, **`8443:443`**, **`env_file`** for **`proxy.env.example`** plus optional **`proxy.env`**, volume mount for Caddyfile, **`host.docker.internal:host-gateway`**, named volumes for Caddy data/config. |
| **Client** | `client/websocket.go` | For **`wss:`**: copy **`DefaultDialer`**, set **`TLSClientConfig`** with **`NextProtos: []string{"http/1.1"`}**, **`InsecureSkipVerify`** when **`--skip-tls-verify`**, **`ServerName`** from URL host with **loopback IPs mapped to `localhost`** for SNI. Narrower detection of “username taken” errors on read (no broad **`Username`** substring). |
| **Server config** | `config/config.go` | **`godotenv.Overload`** instead of **`Load`** for **`config/.env`** so file values override pre-set **`MARCHAT_*`** in the process environment. |
| **Tests** | `config/config_test.go` | **`TestDotenvFileOverridesProcessEnv`** replaces old precedence test: **`.env` must override** conflicting process **`MARCHAT_PORT`**. |
| **Scripts** | `scripts/build-linux.sh` | Local **`CGO_ENABLED=0`** + **ldflags** build for **`marchat-server`** / **`marchat-client`** (defaults **`GOOS=linux`**; override for cross-compile). |
| **Scripts** | `scripts/build-windows.ps1` | Same flags for **`marchat-server.exe`** / **`marchat-client.exe`** on Windows amd64. |
| **Scripts** | `scripts/connect-local-wss.sh` | Unix: loads **`config/.env`**, runs **`marchat-client`** to **`wss://localhost:8443/ws`** with E2E + admin + **`--skip-tls-verify`**. |
| **Scripts** | `scripts/connect-local-wss.ps1` | Windows: same behavior for **`marchat-client.exe`**. |

---

## Debugging checklist

| Symptom | Where to look |
|--------|----------------|
| **`remote error: tls: internal error`** on connect | Caddyfile must use **named hosts** + **`tls internal`**; recreate Caddy volumes if certs were issued under bad config; use **`wss://localhost:8443`**. |
| **Reconnect loop / dial failures** | Client debug log: **Windows** `%APPDATA%\marchat\marchat-client-debug.log`; **Linux** `~/.config/marchat/marchat-client-debug.log`; **macOS** `~/Library/Application Support/marchat/marchat-client-debug.log` (unless **`MARCHAT_CONFIG_DIR`** is set). |
| **Invalid admin key** | Server must use same key as client; with **Overload**, **`config/.env`** overrides stale shell env after **server restart**. |
| **Keystore decrypt error** | Wrong **`--keystore-passphrase`**, corrupted file, or (on older clients) path-dependent keystore salt if **`keystore.dat`** moved; use a current client build (embedded salt + auto-migration). If **`MARCHAT_GLOBAL_E2E_KEY`** is set, the client uses the env key and does not update the file—unset it to use the on-disk key again. Backup/remove **`keystore.dat`** only if you intend to recreate the keystore (you will need the same global key via env or peer copy). |
| **Caddy cannot reach server** | Server on **8080**, Docker **`host.docker.internal`** (Compose **`extra_hosts`**). |

---

## Breaking changes

These apply to the **server** and **client** behavior shipped with the Caddy / WSS work. The **WebSocket wire protocol** between client and server is **unchanged**.

### 1. `config/.env` overrides process environment (`godotenv.Overload`)

**Previous behavior:** `godotenv.Load` did **not** replace variables that were **already set** in the process environment. If `MARCHAT_ADMIN_KEY` (or any other key) was present in the environment before the server read `config/.env`, the **environment value won** and the file was ignored for that key.

**Current behavior:** `godotenv.Overload` applies **`config/.env` on top of** the process environment. Any key present in **`config/.env` overwrites** the same **`MARCHAT_*`** name in the process at server startup.

| Who is affected | What to do |
|-----------------|------------|
| Deployments that **mount or ship** a `config/.env` (or `.env` under `MARCHAT_CONFIG_DIR`) **and** inject overlapping **`MARCHAT_*`** via Docker/Kubernetes/systemd | Treat the **file as authoritative** for those keys, or **remove** conflicting keys from the file if orchestrator secrets must win. |
| Operators who relied on **“env always beats .env”** | Align secrets: either stop shipping a conflicting `.env` or remove the env var and use only the file. |
| Local dev / single `config/.env` as source of truth | No change required; this matches the common expectation that editing the file updates the server after restart. |

**Not a protocol break:** clients do not need an upgrade solely because of this; only **server config precedence** changed.

### 2. Client WebSocket read errors vs “username taken”

**Previous behavior:** Any read error whose message contained the substring **`Username`** (capital U) was classified as a **username conflict** and **did not** trigger the reconnect loop.

**Current behavior:** Only messages matching **case-insensitive** phrases like **`username already taken`**, **`username is already taken`**, or **`duplicate username`** are treated as username errors.

| Who is affected | What to do |
|-----------------|------------|
| Rare: a server or proxy closing the socket with a **custom close reason** that mentioned **`Username`** but was not a duplicate-user case | That case may now be treated as a **generic disconnect** and **reconnect** instead of a static “pick another username” banner. |

### 3. Deploy artifacts (additive, not breaking)

- **`docker-compose.proxy.yml`**, **`deploy/caddy/`**, **`scripts/connect-local-wss.{sh,ps1}`**, and **`scripts/build-{linux.sh,windows.ps1}`** are **optional**; existing **direct `ws://` / `wss://` to marchat** (no Caddy) flows are unchanged.
- **`CADDY-REVERSE-PROXY.md`** is documentation only.

---

## Optional: production hardening

- Terminate TLS with **Let's Encrypt** (or your CA) on Caddy using a **real DNS name**; drop **`--skip-tls-verify`** on clients.
- Set **`MARCHAT_ALLOWED_USERS`** on the server to restrict usernames.
- If you **both** mount **`config/.env`** **and** inject **`MARCHAT_*`** secrets, see **[Breaking changes](#breaking-changes)** - file values override env for keys listed in the file.
