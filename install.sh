#!/bin/bash
# Memory CLI installer
# Usage: curl -sSL https://raw.githubusercontent.com/goflink/memory/main/install.sh | bash

set -e

REPO="goflink/memory"
BINARY_NAME="memory"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    linux|darwin) ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Get latest release
echo "Fetching latest release..."
LATEST=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
    echo "Failed to get latest release"
    exit 1
fi

echo "Installing memory $LATEST for $OS/$ARCH..."

# Download
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST/${BINARY_NAME}_${LATEST#v}_${OS}_${ARCH}.tar.gz"
TMP_DIR=$(mktemp -d)

curl -sL "$DOWNLOAD_URL" | tar xz -C "$TMP_DIR"

# Install
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/"
else
    echo "Need sudo to install to $INSTALL_DIR"
    sudo mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/"
fi

rm -rf "$TMP_DIR"

# Verify
if command -v memory &> /dev/null; then
    echo "memory installed successfully!"
    memory version
else
    echo "Installation complete. Add $INSTALL_DIR to your PATH if not already."
fi
