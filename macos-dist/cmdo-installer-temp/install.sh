#!/bin/bash
set -e

INSTALL_NAME="cmdo"
BASE_URL="https://fd265ec3490d.ngrok-free.app"

echo ""
echo "🚀 CMDO Installer for macOS"
echo ""

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    arm64|aarch64) ARCH="arm64" ;;
    x86_64|amd64) ARCH="amd64" ;;
    *) echo "Error: Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Detect OS
OS=$(uname -s)
case "$OS" in
    Darwin) OS="macos" ;;
    Linux) OS="linux" ;;
    *) echo "Error: Unsupported OS: $OS"; exit 1 ;;
esac

BINARY_NAME="cmdo-${OS}-${ARCH}"
TEMP_FILE="/tmp/${BINARY_NAME}.$$"

echo "→ System: ${OS} (${ARCH})"
echo "→ Downloading ${BINARY_NAME}..."

# Download with ngrok-skip-browser-warning header
if ! curl -fsSL -H "ngrok-skip-browser-warning: true" "${BASE_URL}/${BINARY_NAME}" -o "$TEMP_FILE"; then
    echo "Error: Download failed!"
    rm -f "$TEMP_FILE"
    exit 1
fi

# Verify it's a binary
if ! file "$TEMP_FILE" | grep -q "executable\|Mach-O"; then
    echo "Error: Downloaded file is not a valid binary"
    rm -f "$TEMP_FILE"
    exit 1
fi

# Remove quarantine (macOS only)
if [ "$OS" = "macos" ]; then
    echo "→ Removing quarantine attributes..."
    xattr -cr "$TEMP_FILE" 2>/dev/null || true
    xattr -d com.apple.quarantine "$TEMP_FILE" 2>/dev/null || true
fi

# Make executable
chmod +x "$TEMP_FILE"

# Find install directory
INSTALL_DIR="$HOME/.local/bin"
if [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    mkdir -p "$INSTALL_DIR"
fi

echo "→ Installing to ${INSTALL_DIR}..."

# Install
if [ "$INSTALL_DIR" = "/usr/local/bin" ] && [ ! -w "$INSTALL_DIR" ]; then
    if ! sudo cp "$TEMP_FILE" "$INSTALL_DIR/$INSTALL_NAME"; then
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        cp "$TEMP_FILE" "$INSTALL_DIR/$INSTALL_NAME"
    fi
    sudo chmod +x "$INSTALL_DIR/$INSTALL_NAME" 2>/dev/null || chmod +x "$INSTALL_DIR/$INSTALL_NAME"
else
    cp "$TEMP_FILE" "$INSTALL_DIR/$INSTALL_NAME"
    chmod +x "$INSTALL_DIR/$INSTALL_NAME"
fi

# Remove quarantine from installed binary
if [ "$OS" = "macos" ]; then
    xattr -cr "$INSTALL_DIR/$INSTALL_NAME" 2>/dev/null || true
fi

# Cleanup
rm -f "$TEMP_FILE"

echo ""
echo "✅ CMDO installed to ${INSTALL_DIR}/${INSTALL_NAME}"

# Check PATH
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        echo ""
        echo "⚠️  Add to PATH:"
        echo "export PATH=\"${INSTALL_DIR}:\$PATH\""
        echo ""
        ;;
esac

echo "Run: cmdo setup"
echo ""