# Local Windows amd64 build with the same flags as release (CGO off, version ldflags).
# Linux/macOS host: use ./scripts/build-linux.sh (set GOOS=darwin for macOS).
$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)

$VERSION = "v0.11.0-beta.5"
$BUILD_TIME = (Get-Date).ToUniversalTime().ToString("o")
$GIT_COMMIT = git rev-parse --short HEAD 2>$null
if (-not $GIT_COMMIT) { $GIT_COMMIT = "unknown" }

$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"

$ldflags = @(
    "-X", "github.com/Cod-e-Codes/marchat/shared.ClientVersion=$VERSION",
    "-X", "github.com/Cod-e-Codes/marchat/shared.ServerVersion=$VERSION",
    "-X", "github.com/Cod-e-Codes/marchat/shared.BuildTime=$BUILD_TIME",
    "-X", "github.com/Cod-e-Codes/marchat/shared.GitCommit=$GIT_COMMIT"
) -join " "

Write-Host "Building marchat $VERSION (CGO_ENABLED=0)..." -ForegroundColor Green
go mod tidy
go build -ldflags $ldflags -o marchat-server.exe ./cmd/server
go build -ldflags $ldflags -o marchat-client.exe ./client
Write-Host "Done: marchat-server.exe, marchat-client.exe" -ForegroundColor Green
