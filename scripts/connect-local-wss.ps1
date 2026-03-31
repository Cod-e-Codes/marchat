# Connect marchat-client to local Caddy (8443) + E2E, reading secrets from config\.env.
# Usage: .\scripts\connect-local-wss.ps1   (Unix: ./scripts/connect-local-wss.sh)
# Optional: -KeystorePass "yourpass"  (default: prompt)
#
# Client username for --username: if config\.env defines MARCHAT_CLIENT_USERNAME, that value
# is used; otherwise the first comma-separated name in MARCHAT_USERS is used (same file).

param([string]$KeystorePass = "")

$ErrorActionPreference = "Stop"
$root = Split-Path $PSScriptRoot -Parent
Set-Location $root

$envPath = Join-Path $root "config\.env"
if (-not (Test-Path $envPath)) { throw "Missing $envPath - create it with MARCHAT_ADMIN_KEY, MARCHAT_USERS, MARCHAT_GLOBAL_E2E_KEY" }

$clientUsername = $null
$usersLine = $null
foreach ($line in Get-Content $envPath) {
  if ($line -match '^MARCHAT_GLOBAL_E2E_KEY=(.+)$') { $env:MARCHAT_GLOBAL_E2E_KEY = $matches[1].Trim() }
  if ($line -match '^MARCHAT_ADMIN_KEY=(.+)$') { $adminKey = $matches[1].Trim() }
  if ($line -match '^MARCHAT_CLIENT_USERNAME=(.+)$') { $clientUsername = $matches[1].Trim() }
  if ($line -match '^MARCHAT_USERS=(.+)$') { $usersLine = $matches[1].Trim() }
}

if (-not $adminKey) { throw "MARCHAT_ADMIN_KEY not found in config\.env" }
if (-not $env:MARCHAT_GLOBAL_E2E_KEY) { throw "MARCHAT_GLOBAL_E2E_KEY not found in config\.env" }

if (-not $clientUsername) {
  if ($usersLine) {
    $clientUsername = ($usersLine -split ',')[0].Trim()
  }
}
if (-not $clientUsername) { throw "Set MARCHAT_CLIENT_USERNAME or MARCHAT_USERS in config\.env" }

if (-not $KeystorePass) {
  $sec = Read-Host -AsSecureString "Keystore passphrase"
  $ptr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($sec)
  try { $KeystorePass = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($ptr) }
  finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($ptr) }
}

$exe = Join-Path $root "marchat-client.exe"
if (-not (Test-Path $exe)) { throw "Build client first: .\scripts\build-windows.ps1 (Unix/macOS: ./scripts/build-linux.sh; set GOOS=darwin for macOS)" }

# Always use hostname localhost so TLS SNI matches Caddy internal cert.
& $exe `
  --server "wss://localhost:8443/ws" `
  --username $clientUsername `
  --admin `
  --admin-key $adminKey `
  --e2e `
  --keystore-passphrase $KeystorePass `
  --skip-tls-verify
