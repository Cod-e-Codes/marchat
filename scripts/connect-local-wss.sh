#!/usr/bin/env bash
# Connect marchat-client to local Caddy (8443) + E2E, reading secrets from config/.env.
# Usage: ./scripts/connect-local-wss.sh
# Optional: KEYSTORE_PASS=yourpass ./scripts/connect-local-wss.sh  (else prompts)
#
# Client username for --username: if config/.env defines MARCHAT_CLIENT_USERNAME, that value
# is used; otherwise the first comma-separated name in MARCHAT_USERS is used (same file).

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ENV_FILE="${ROOT}/config/.env"
if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Missing ${ENV_FILE} - create it with MARCHAT_ADMIN_KEY, MARCHAT_USERS, MARCHAT_GLOBAL_E2E_KEY" >&2
  exit 1
fi

trim() {
  local s="$1"
  s="${s#"${s%%[![:space:]]*}"}"
  s="${s%"${s##*[![:space:]]}"}"
  printf '%s' "$s"
}

admin_key=""
client_username=""
users_line=""
while IFS= read -r line || [[ -n "${line}" ]]; do
  line="${line%$'\r'}"
  [[ "${line}" =~ ^[[:space:]]*# ]] && continue
  [[ -z "${line// }" ]] && continue
  [[ "${line}" != *=* ]] && continue
  key="${line%%=*}"
  val="${line#*=}"
  key="$(trim "${key}")"
  val="$(trim "${val}")"
  case "${key}" in
    MARCHAT_GLOBAL_E2E_KEY) export MARCHAT_GLOBAL_E2E_KEY="${val}" ;;
    MARCHAT_ADMIN_KEY) admin_key="${val}" ;;
    MARCHAT_CLIENT_USERNAME) client_username="${val}" ;;
    MARCHAT_USERS) users_line="${val}" ;;
  esac
done < "${ENV_FILE}"

if [[ -z "${admin_key}" ]]; then
  echo "MARCHAT_ADMIN_KEY not found in config/.env" >&2
  exit 1
fi
if [[ -z "${MARCHAT_GLOBAL_E2E_KEY:-}" ]]; then
  echo "MARCHAT_GLOBAL_E2E_KEY not found in config/.env" >&2
  exit 1
fi

if [[ -z "${client_username}" && -n "${users_line}" ]]; then
  IFS=',' read -r first _ <<< "${users_line}"
  client_username="$(trim "${first}")"
fi
if [[ -z "${client_username}" ]]; then
  echo "Set MARCHAT_CLIENT_USERNAME or MARCHAT_USERS in config/.env" >&2
  exit 1
fi

if [[ -z "${KEYSTORE_PASS:-}" ]]; then
  read -r -s -p "Keystore passphrase: " KEYSTORE_PASS
  echo
fi

CLIENT="${ROOT}/marchat-client"
if [[ ! -f "${CLIENT}" ]]; then
  echo "Build client first: ./scripts/build-linux.sh" >&2
  exit 1
fi

# Always use hostname localhost so TLS SNI matches Caddy internal cert.
exec "${CLIENT}" \
  --server "wss://localhost:8443/ws" \
  --username "${client_username}" \
  --admin \
  --admin-key "${admin_key}" \
  --e2e \
  --keystore-passphrase "${KEYSTORE_PASS}" \
  --skip-tls-verify
