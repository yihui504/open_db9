#!/bin/bash
# Open-DB9 Installation Script
# Supports: Linux, macOS (with Homebrew)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
REPO_URL="https://github.com/open-db9/db9/releases"

# Detect OS
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux*)
        OS="linux"
        ;;
    Darwin*)
        OS="darwin"
        ;;
    *)
        echo -e "${RED}Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo -e "${GREEN}Installing Open-DB9 for $OS-$ARCH...${NC}"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${YELLOW}Note: Installation requires root privileges. You may be prompted for your password.${NC}"
    SUDO="sudo"
else
    SUDO=""
fi

# Create temporary directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Download binaries
echo "Downloading binaries..."
BINARY_URL="$REPO_URL/download/$VERSION/db9-$OS-$ARCH"
curl -L "$BINARY_URL" -o "$TMP_DIR/db9" || {
    echo -e "${RED}Failed to download binary from $BINARY_URL${NC}"
    echo -e "${YELLOW}Please install manually from: $REPO_URL${NC}"
    exit 1
}

# Verify binary
echo "Verifying binary..."
chmod +x "$TMP_DIR/db9"

# Install binary
echo "Installing to $INSTALL_DIR..."
$SUDO mkdir -p "$INSTALL_DIR"
$SUDO cp "$TMP_DIR/db9" "$INSTALL_DIR/db9"

# Create configuration directory
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/db9"
mkdir -p "$CONFIG_DIR"

# Create data directory
DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/db9"
mkdir -p "$DATA_DIR"

echo -e "${GREEN}Installation complete!${NC}"
echo ""
echo "To get started:"
echo "  1. Run: db9 --help"
echo "  2. Initialize a database: db9 init"
echo ""
echo "For more information, visit: https://github.com/open-db9/db9"
