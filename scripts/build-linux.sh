#!/usr/bin/env bash
# Local Linux build with the same flags as release (CGO off, version ldflags).
# Usage: ./scripts/build-linux.sh
# Optional: GOARCH=arm64 GOOS=linux ./scripts/build-linux.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VERSION="v0.10.0-beta.2"
BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"

export CGO_ENABLED=0
export GOOS="${GOOS:-linux}"
export GOARCH="${GOARCH:-amd64}"

ldflags="-X github.com/Cod-e-Codes/marchat/shared.ClientVersion=${VERSION} -X github.com/Cod-e-Codes/marchat/shared.ServerVersion=${VERSION} -X github.com/Cod-e-Codes/marchat/shared.BuildTime=${BUILD_TIME} -X github.com/Cod-e-Codes/marchat/shared.GitCommit=${GIT_COMMIT}"

echo "Building marchat ${VERSION} (CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH})..."
go mod tidy
go build -ldflags "${ldflags}" -o marchat-server ./cmd/server
go build -ldflags "${ldflags}" -o marchat-client ./client
echo "Done: marchat-server, marchat-client"
