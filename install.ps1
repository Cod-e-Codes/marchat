#!/usr/bin/env pwsh

# Cross-platform install script for marchat (PowerShell version)
# Supports Windows, Linux, macOS, and Android (via PowerShell Core)

param(
    [string]$Version = "v1.0.0"
)

$ErrorActionPreference = "Stop"

Write-Host "marchat installer (PowerShell)" -ForegroundColor Green
Write-Host ""

# Detect OS and architecture
if ($IsWindows -or $env:OS -eq "Windows_NT") {
    $OS = "windows"
} elseif ($IsLinux) {
    $OS = "linux"
} elseif ($IsMacOS) {
    $OS = "darwin"
} else {
    # Fallback detection
    $unameOutput = & uname
    switch ($unameOutput.ToLower()) {
        "linux" { $OS = "linux" }
        "darwin" { $OS = "darwin" }
        default { 
            Write-Host "[ERROR] Error: Unsupported OS: $unameOutput" -ForegroundColor Red
            exit 1
        }
    }
}

# Detect architecture
if ([System.Environment]::Is64BitOperatingSystem) {
    if ($IsWindows -or $env:OS -eq "Windows_NT") {
        $ARCH = "amd64"
    } else {
        $unameArch = & uname -m
        switch ($unameArch) {
            "x86_64" { $ARCH = "amd64" }
            "aarch64" { $ARCH = "arm64" }
            "armv7l" { $ARCH = "arm" }
            default { $ARCH = "amd64" }
        }
    }
} else {
    $ARCH = "386"
}

# Termux: use linux-arm64 release asset (same static binary as Linux aarch64; not android-* zip)
if ($env:PREFIX -and $env:PREFIX -like "*com.termux*") {
    $OS = "linux"
    $ARCH = "arm64"
}

# Construct GitHub release URL
$URL = "https://github.com/Cod-e-Codes/marchat/releases/download/$Version/marchat-$Version-$OS-$ARCH.zip"

# Create temporary directories
$TEMP_DIR = [System.IO.Path]::GetTempPath() + [System.Guid]::NewGuid().ToString()
$ZIP_FILE = Join-Path $TEMP_DIR "marchat.zip"
$EXTRACT_DIR = Join-Path $TEMP_DIR "extracted"

Write-Host "[INFO] Detected OS: $OS" -ForegroundColor Cyan
Write-Host "[INFO] Detected ARCH: $ARCH" -ForegroundColor Cyan
Write-Host "[INFO] Download URL: $URL" -ForegroundColor Cyan
Write-Host "[INFO] Temp directory: $TEMP_DIR" -ForegroundColor Cyan
Write-Host ""

# Create temp directories
New-Item -ItemType Directory -Path $TEMP_DIR -Force | Out-Null
New-Item -ItemType Directory -Path $EXTRACT_DIR -Force | Out-Null

# Download the zip
Write-Host "[INFO] Downloading marchat $Version..." -ForegroundColor Green
try {
    Invoke-WebRequest -Uri $URL -OutFile $ZIP_FILE -UseBasicParsing
} catch {
    Write-Host "[ERROR] Download failed: $($_.Exception.Message)" -ForegroundColor Red
    Remove-Item -Path $TEMP_DIR -Recurse -Force -ErrorAction SilentlyContinue
    exit 1
}

# Extract zip
Write-Host "[INFO] Extracting..." -ForegroundColor Green
try {
    Expand-Archive -Path $ZIP_FILE -DestinationPath $EXTRACT_DIR -Force
} catch {
    Write-Host "[ERROR] Extraction failed: $($_.Exception.Message)" -ForegroundColor Red
    Remove-Item -Path $TEMP_DIR -Recurse -Force -ErrorAction SilentlyContinue
    exit 1
}

# Determine install directory based on OS
$INSTALL_DIR = ""
$CONFIG_DIR = ""
$USE_SUDO = $false

switch ($OS) {
    "linux" {
        # Check if we're in Termux (Android)
        if ($env:PREFIX -and $env:PREFIX -like "*com.termux*") {
            $INSTALL_DIR = "$env:PREFIX/bin"
            $CONFIG_DIR = "$env:HOME/.config/marchat"
            $USE_SUDO = $false
        } else {
            # Regular Linux
            $INSTALL_DIR = "/usr/local/bin"
            $CONFIG_DIR = "$env:HOME/.config/marchat"
            $USE_SUDO = $true
        }
    }
    "android" {
        $INSTALL_DIR = "$env:PREFIX/bin"
        $CONFIG_DIR = "$env:HOME/.config/marchat"
        $USE_SUDO = $false
    }
    "darwin" {
        $INSTALL_DIR = "/usr/local/bin"
        $CONFIG_DIR = "$env:HOME/Library/Application Support/marchat"
        $USE_SUDO = $true
    }
    "windows" {
        # For Windows, install to user's local bin directory
        $localBin = "$env:USERPROFILE\.local\bin"
        $INSTALL_DIR = $localBin
        $CONFIG_DIR = "$env:APPDATA\marchat"
        $USE_SUDO = $false
    }
    default {
        Write-Host "[ERROR] Unsupported OS: $OS" -ForegroundColor Red
        Remove-Item -Path $TEMP_DIR -Recurse -Force -ErrorAction SilentlyContinue
        exit 1
    }
}

Write-Host "[INFO] Installing to: $INSTALL_DIR" -ForegroundColor Yellow
Write-Host "[INFO] Config directory: $CONFIG_DIR" -ForegroundColor Yellow
Write-Host ""

# Create install directory
if (!(Test-Path $INSTALL_DIR)) {
    if ($USE_SUDO -and !$IsWindows) {
        Write-Host "[INFO] Creating install directory (requires sudo)..." -ForegroundColor Yellow
        & sudo mkdir -p $INSTALL_DIR
    } else {
        New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
    }
}

# Find the correct binary files
$SERVER_BINARY = Get-ChildItem -Path $EXTRACT_DIR | Where-Object { $_.Name -like "*marchat-server*" } | Select-Object -First 1 -ExpandProperty FullName
$CLIENT_BINARY = Get-ChildItem -Path $EXTRACT_DIR | Where-Object { $_.Name -like "*marchat-client*" } | Select-Object -First 1 -ExpandProperty FullName

if ([string]::IsNullOrEmpty($SERVER_BINARY) -or [string]::IsNullOrEmpty($CLIENT_BINARY)) {
    Write-Host "[ERROR] Could not find marchat binaries in the downloaded archive" -ForegroundColor Red
    Write-Host "[INFO] Contents of extract directory:" -ForegroundColor Yellow
    Get-ChildItem -Path $EXTRACT_DIR | Format-Table Name, Length, LastWriteTime
    Remove-Item -Path $TEMP_DIR -Recurse -Force -ErrorAction SilentlyContinue
    exit 1
}

# Copy binaries
Write-Host "[INFO] Copying binaries..." -ForegroundColor Green

$serverDest = Join-Path $INSTALL_DIR "marchat-server"
$clientDest = Join-Path $INSTALL_DIR "marchat-client"

# Add .exe extension on Windows
if ($OS -eq "windows") {
    $serverDest += ".exe"
    $clientDest += ".exe"
}

try {
    if ($USE_SUDO -and !$IsWindows) {
        & sudo cp $SERVER_BINARY $serverDest
        & sudo cp $CLIENT_BINARY $clientDest
        & sudo chmod +x $serverDest $clientDest
    } else {
        Copy-Item -Path $SERVER_BINARY -Destination $serverDest -Force
        Copy-Item -Path $CLIENT_BINARY -Destination $clientDest -Force
        
        # Make executable on Unix-like systems
        if (!$IsWindows) {
            & chmod +x $serverDest $clientDest
        }
    }
} catch {
    Write-Host "[ERROR] Error copying binaries: $($_.Exception.Message)" -ForegroundColor Red
    Remove-Item -Path $TEMP_DIR -Recurse -Force -ErrorAction SilentlyContinue
    exit 1
}

# Create config directory
if (!(Test-Path $CONFIG_DIR)) {
    New-Item -ItemType Directory -Path $CONFIG_DIR -Force | Out-Null
}

# Add to PATH on Windows if not already there
if ($OS -eq "windows") {
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($currentPath -notlike "*$INSTALL_DIR*") {
        Write-Host "[INFO] Adding $INSTALL_DIR to user PATH..." -ForegroundColor Yellow
        [Environment]::SetEnvironmentVariable("PATH", "$currentPath;$INSTALL_DIR", "User")
        Write-Host "[WARN] You may need to restart your terminal for PATH changes to take effect" -ForegroundColor Yellow
    }
}

# Clean up temp directory
Write-Host "[INFO] Cleaning up..." -ForegroundColor Green
Remove-Item -Path $TEMP_DIR -Recurse -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "[OK] Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "[INFO] Binaries installed to: $INSTALL_DIR" -ForegroundColor Cyan
Write-Host "[INFO] Config directory: $CONFIG_DIR" -ForegroundColor Cyan

if ($OS -eq "windows") {
    Write-Host ""
    Write-Host "Quick start:" -ForegroundColor Yellow
    Write-Host "  1. Start server: marchat-server" -ForegroundColor White
    Write-Host "  2. Connect client: marchat-client --username yourname" -ForegroundColor White
    Write-Host ""
    Write-Host "[TIP] If commands are not found, restart your terminal or run:" -ForegroundColor Yellow
    Write-Host "   refreshenv" -ForegroundColor White
} else {
    Write-Host ""
    Write-Host "Quick start:" -ForegroundColor Yellow
    Write-Host "  1. Start server: marchat-server" -ForegroundColor White
    Write-Host "  2. Connect client: marchat-client --username yourname" -ForegroundColor White
}

Write-Host ""
Write-Host "For more information, visit: https://github.com/Cod-e-Codes/marchat" -ForegroundColor Blue
