#!/usr/bin/env bash

# Cross-platform install script for marchat
# Supports Linux, macOS, Windows (via Git Bash/WSL), and Android (Termux)

set -e  # Exit on any error

VERSION="v0.9.0-beta.5"

# Detect OS and architecture
OS=$(uname | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map uname output to Go arch naming
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  armv7l) ARCH="arm" ;;
  i386|i686) ARCH="386" ;;
esac

# Handle Windows detection
if [[ "$OS" == *"msys"* ]] || [[ "$OS" == *"mingw"* ]] || [[ "$OS" == *"cygwin"* ]]; then
  OS="windows"
fi

# Construct GitHub release URL
URL="https://github.com/Cod-e-Codes/marchat/releases/download/$VERSION/marchat-$VERSION-$OS-$ARCH.zip"

# Create temporary directories
TEMP_DIR=$(mktemp -d)
ZIP_FILE="$TEMP_DIR/marchat.zip"
EXTRACT_DIR="$TEMP_DIR/extracted"

echo "🔍 Detected OS: $OS"
echo "🔍 Detected ARCH: $ARCH"
echo "📥 Download URL: $URL"
echo "📁 Temp directory: $TEMP_DIR"

# Check if curl is available
if ! command -v curl &> /dev/null; then
  echo "❌ Error: curl is required but not installed"
  exit 1
fi

# Check if unzip is available
if ! command -v unzip &> /dev/null; then
  echo "❌ Error: unzip is required but not installed"
  exit 1
fi

# Download the zip
echo "📥 Downloading marchat $VERSION..."
curl -L -o "$ZIP_FILE" "$URL"
if [ $? -ne 0 ]; then
  echo "❌ Download failed!"
  exit 1
fi

# Extract zip
mkdir -p "$EXTRACT_DIR"
echo "📦 Extracting..."
unzip -q "$ZIP_FILE" -d "$EXTRACT_DIR"

# Determine install directory based on OS
INSTALL_DIR=""
CONFIG_DIR=""
USE_SUDO=""

case "$OS" in
  linux|android)
    # Check if we're in Termux (Android)
    if [[ -n "$PREFIX" && "$PREFIX" == *"com.termux"* ]]; then
      INSTALL_DIR="$PREFIX/bin"
      CONFIG_DIR="$HOME/.config/marchat"
      USE_SUDO=""
    else
      # Regular Linux
      INSTALL_DIR="/usr/local/bin"
      CONFIG_DIR="$HOME/.config/marchat"
      USE_SUDO="sudo"
    fi
    ;;
  darwin)
    INSTALL_DIR="/usr/local/bin"
    CONFIG_DIR="$HOME/Library/Application Support/marchat"
    USE_SUDO="sudo"
    ;;
  windows)
    # For Windows, install to user's local bin directory
    INSTALL_DIR="$HOME/.local/bin"
    CONFIG_DIR="$APPDATA/marchat"
    USE_SUDO=""
    ;;
  *)
    echo "❌ Unsupported OS: $OS"
    exit 1
    ;;
esac

echo "📁 Installing to: $INSTALL_DIR"
echo "⚙️  Config directory: $CONFIG_DIR"

# Create install directory
if [[ -n "$USE_SUDO" ]]; then
  $USE_SUDO mkdir -p "$INSTALL_DIR"
else
  mkdir -p "$INSTALL_DIR"
fi

# Find the correct binary files
SERVER_BINARY=""
CLIENT_BINARY=""

for file in "$EXTRACT_DIR"/*; do
  if [[ "$file" == *"marchat-server"* ]]; then
    SERVER_BINARY="$file"
  elif [[ "$file" == *"marchat-client"* ]]; then
    CLIENT_BINARY="$file"
  fi
done

if [[ -z "$SERVER_BINARY" ]] || [[ -z "$CLIENT_BINARY" ]]; then
  echo "❌ Error: Could not find marchat binaries in the downloaded archive"
  echo "📁 Contents of extract directory:"
  ls -la "$EXTRACT_DIR"
  exit 1
fi

# Copy binaries
echo "📋 Copying binaries..."
if [[ -n "$USE_SUDO" ]]; then
  $USE_SUDO cp "$SERVER_BINARY" "$INSTALL_DIR/marchat-server"
  $USE_SUDO cp "$CLIENT_BINARY" "$INSTALL_DIR/marchat-client"
  $USE_SUDO chmod +x "$INSTALL_DIR/marchat-server" "$INSTALL_DIR/marchat-client"
else
  cp "$SERVER_BINARY" "$INSTALL_DIR/marchat-server"
  cp "$CLIENT_BINARY" "$INSTALL_DIR/marchat-client"
  chmod +x "$INSTALL_DIR/marchat-server" "$INSTALL_DIR/marchat-client"
fi

# Create config directory
mkdir -p "$CONFIG_DIR"

# Clean up temp directory
echo "🧹 Cleaning up..."
rm -rf "$TEMP_DIR"

echo ""
echo "✅ Installation complete!"
echo ""
echo "📁 Binaries installed to: $INSTALL_DIR"
echo "⚙️  Config directory: $CONFIG_DIR"
echo ""
echo "🚀 Quick start:"
echo "  1. Start server: marchat-server"
echo "  2. Connect client: marchat-client --username yourname"
echo ""
echo "📖 For more information, visit: https://github.com/Cod-e-Codes/marchat"
