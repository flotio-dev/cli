#!/bin/sh
set -eu

# Flotio CLI installer — one-liner:
#   curl -fsSL https://raw.githubusercontent.com/flotio-dev/cli/main/install.sh | sh

REPO="flotio-dev/cli"
BINARY="flotio"

# --- Detect OS ---
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  linux)   GOOS="linux" ;;
  darwin)  GOOS="darwin" ;;
  *)
    # Try MSYS/MinGW/Cygwin on Windows
    if uname -s | grep -qi "mingw\|msys\|cygwin"; then
      GOOS="windows"
    else
      echo "Unsupported OS: $OS" >&2
      exit 1
    fi
    ;;
esac

case "$ARCH" in
  x86_64|amd64) GOARCH="amd64" ;;
  aarch64|arm64) GOARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# --- Fetch latest release ---
echo "Fetching latest release for ${GOOS}-${GOARCH}..."
RELEASE_URL="https://github.com/${REPO}/releases/latest/download/flotio-${GOOS}-${GOARCH}"
if [ "$GOOS" = "windows" ]; then
  RELEASE_URL="${RELEASE_URL}.exe"
fi

# --- Download ---
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

DEST="$TMPDIR/$BINARY"
if ! curl -fsSL -o "$DEST" "$RELEASE_URL"; then
  echo "Failed to download $RELEASE_URL" >&2
  echo "Make sure a release exists with tag v0.1.0 or later." >&2
  exit 1
fi
chmod +x "$DEST"

# --- Install ---
INSTALL_DIR=""
if [ -w /usr/local/bin ]; then
  INSTALL_DIR="/usr/local/bin"
elif command -v sudo >/dev/null 2>&1; then
  INSTALL_DIR="/usr/local/bin"
  echo "Installing to /usr/local/bin (sudo required)..."
  sudo mv "$DEST" "$INSTALL_DIR/$BINARY"
  echo "✓ flotio installed to /usr/local/bin/flotio"
  exit 0
else
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

mv "$DEST" "$INSTALL_DIR/$BINARY"
echo "✓ flotio installed to $INSTALL_DIR/flotio"

# --- PATH check ---
if ! echo "$PATH" | tr ':' '\n' | grep -qxF "$INSTALL_DIR"; then
  echo ""
  echo "⚠ $INSTALL_DIR is not in your PATH."
  echo "  Add this to your ~/.bashrc or ~/.zshrc:"
  echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
fi
