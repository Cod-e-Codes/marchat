# Build script for marchat v0.9.0-beta.5
# This script builds all platform targets and creates release zips

$ErrorActionPreference = "Stop"

$VERSION = "v0.9.0-beta.5"
$BUILD_DIR = "build"
$RELEASE_DIR = "release"

Write-Host "Building marchat $VERSION..." -ForegroundColor Green

# Create build and release directories
New-Item -ItemType Directory -Force -Path $BUILD_DIR | Out-Null
New-Item -ItemType Directory -Force -Path $RELEASE_DIR | Out-Null

# Clean previous builds
if (Test-Path $BUILD_DIR) { Remove-Item "$BUILD_DIR\*" -Recurse -Force }
if (Test-Path $RELEASE_DIR) { Remove-Item "$RELEASE_DIR\*" -Recurse -Force }

# Get current timestamp and git commit
$BUILD_TIME = (Get-Date).ToUniversalTime().ToString('o')
$GIT_COMMIT = git rev-parse --short HEAD 2>$null
if (-not $GIT_COMMIT) { $GIT_COMMIT = "unknown" }

# Build targets with version information
Write-Host "Building for Linux AMD64..." -ForegroundColor Yellow
$env:GOOS = "linux"; $env:GOARCH = "amd64"
go build -ldflags "-X github.com/Cod-e-Codes/marchat/shared.ClientVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.ServerVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.BuildTime='$BUILD_TIME' -X github.com/Cod-e-Codes/marchat/shared.GitCommit=$GIT_COMMIT" -o "$BUILD_DIR/marchat-client-linux-amd64" ./client
go build -ldflags "-X github.com/Cod-e-Codes/marchat/shared.ClientVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.ServerVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.BuildTime='$BUILD_TIME' -X github.com/Cod-e-Codes/marchat/shared.GitCommit=$GIT_COMMIT" -o "$BUILD_DIR/marchat-server-linux-amd64" ./cmd/server

Write-Host "Building for Windows AMD64..." -ForegroundColor Yellow
$env:GOOS = "windows"; $env:GOARCH = "amd64"
go build -ldflags "-X github.com/Cod-e-Codes/marchat/shared.ClientVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.ServerVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.BuildTime='$BUILD_TIME' -X github.com/Cod-e-Codes/marchat/shared.GitCommit=$GIT_COMMIT" -o "$BUILD_DIR/marchat-client-windows-amd64.exe" ./client
go build -ldflags "-X github.com/Cod-e-Codes/marchat/shared.ClientVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.ServerVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.BuildTime='$BUILD_TIME' -X github.com/Cod-e-Codes/marchat/shared.GitCommit=$GIT_COMMIT" -o "$BUILD_DIR/marchat-server-windows-amd64.exe" ./cmd/server

Write-Host "Building for Darwin AMD64..." -ForegroundColor Yellow
$env:GOOS = "darwin"; $env:GOARCH = "amd64"
go build -ldflags "-X github.com/Cod-e-Codes/marchat/shared.ClientVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.ServerVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.BuildTime='$BUILD_TIME' -X github.com/Cod-e-Codes/marchat/shared.GitCommit=$GIT_COMMIT" -o "$BUILD_DIR/marchat-client-darwin-amd64" ./client
go build -ldflags "-X github.com/Cod-e-Codes/marchat/shared.ClientVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.ServerVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.BuildTime='$BUILD_TIME' -X github.com/Cod-e-Codes/marchat/shared.GitCommit=$GIT_COMMIT" -o "$BUILD_DIR/marchat-server-darwin-amd64" ./cmd/server

Write-Host "Building for Android ARM64..." -ForegroundColor Yellow
$env:GOOS = "android"; $env:GOARCH = "arm64"
go build -ldflags "-X github.com/Cod-e-Codes/marchat/shared.ClientVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.ServerVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.BuildTime='$BUILD_TIME' -X github.com/Cod-e-Codes/marchat/shared.GitCommit=$GIT_COMMIT" -o "$BUILD_DIR/marchat-client-android-arm64" ./client
go build -ldflags "-X github.com/Cod-e-Codes/marchat/shared.ClientVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.ServerVersion=$VERSION -X github.com/Cod-e-Codes/marchat/shared.BuildTime='$BUILD_TIME' -X github.com/Cod-e-Codes/marchat/shared.GitCommit=$GIT_COMMIT" -o "$BUILD_DIR/marchat-server-android-arm64" ./cmd/server

# Create release zips
Write-Host "Creating release zips..." -ForegroundColor Yellow

# Linux AMD64
Compress-Archive -Path "$BUILD_DIR/marchat-client-linux-amd64", "$BUILD_DIR/marchat-server-linux-amd64" -DestinationPath "$RELEASE_DIR/marchat-$VERSION-linux-amd64.zip" -Force

# Windows AMD64
Compress-Archive -Path "$BUILD_DIR/marchat-client-windows-amd64.exe", "$BUILD_DIR/marchat-server-windows-amd64.exe" -DestinationPath "$RELEASE_DIR/marchat-$VERSION-windows-amd64.zip" -Force

# Darwin AMD64
Compress-Archive -Path "$BUILD_DIR/marchat-client-darwin-amd64", "$BUILD_DIR/marchat-server-darwin-amd64" -DestinationPath "$RELEASE_DIR/marchat-$VERSION-darwin-amd64.zip" -Force

# Android ARM64
Compress-Archive -Path "$BUILD_DIR/marchat-client-android-arm64", "$BUILD_DIR/marchat-server-android-arm64" -DestinationPath "$RELEASE_DIR/marchat-$VERSION-android-arm64.zip" -Force

Write-Host "Build complete!" -ForegroundColor Green
Write-Host "Release files created in $RELEASE_DIR/:" -ForegroundColor Cyan
Get-ChildItem $RELEASE_DIR | Format-Table Name, Length, LastWriteTime

Write-Host ""
Write-Host "Release assets for ${VERSION}:" -ForegroundColor Magenta
Write-Host "- marchat-$VERSION-linux-amd64.zip"
Write-Host "- marchat-$VERSION-windows-amd64.zip"
Write-Host "- marchat-$VERSION-darwin-amd64.zip"
Write-Host "- marchat-$VERSION-android-arm64.zip"

# Build and push Docker image
Write-Host "Building Docker image for marchat..." -ForegroundColor Yellow

docker build -t codecodesxyz/marchat:$VERSION `
    --build-arg GIT_COMMIT=$GIT_COMMIT `
    --build-arg BUILD_TIME=$BUILD_TIME `
    --build-arg VERSION=$VERSION `
    .

# Tag as latest
docker tag codecodesxyz/marchat:$VERSION codecodesxyz/marchat:latest

# Push images
docker push codecodesxyz/marchat:$VERSION
docker push codecodesxyz/marchat:latest

Write-Host "Docker image build and push complete!" -ForegroundColor Green
