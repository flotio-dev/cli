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
trap 'rm -rf "$TMPDIR"' EXIT INT TERM HUP

DEST="$TMPDIR/$BINARY"
if ! curl -fsSL -o "$DEST" "$RELEASE_URL"; then
  echo "Failed to download $RELEASE_URL" >&2
  echo "Make sure a release exists with tag v0.1.0 or later." >&2
  exit 1
fi
chmod +x "$DEST"

# --- Determine Install Directory ---
INSTALL_DIR="${FLOTIO_INSTALL_DIR:-${INSTALL_DIR:-}}"

if [ -n "$INSTALL_DIR" ]; then
  if mkdir -p "$INSTALL_DIR" 2>/dev/null && [ -w "$INSTALL_DIR" ]; then
    mv "$DEST" "$INSTALL_DIR/$BINARY"
  elif command -v sudo >/dev/null 2>&1; then
    echo "Installing to $INSTALL_DIR (sudo required)..."
    sudo mkdir -p "$INSTALL_DIR"
    sudo mv "$DEST" "$INSTALL_DIR/$BINARY"
    sudo chmod +x "$INSTALL_DIR/$BINARY"
  else
    echo "Cannot write to $INSTALL_DIR and sudo is not available." >&2
    exit 1
  fi
elif [ "$(id -u 2>/dev/null || true)" = "0" ]; then
  INSTALL_DIR="/usr/local/bin"
  mkdir -p "$INSTALL_DIR"
  mv "$DEST" "$INSTALL_DIR/$BINARY"
elif [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
  INSTALL_DIR="/usr/local/bin"
  mv "$DEST" "$INSTALL_DIR/$BINARY"
elif command -v sudo >/dev/null 2>&1; then
  INSTALL_DIR="/usr/local/bin"
  echo "Installing to /usr/local/bin (sudo required)..."
  if sudo mkdir -p "$INSTALL_DIR" && sudo mv "$DEST" "$INSTALL_DIR/$BINARY"; then
    sudo chmod +x "$INSTALL_DIR/$BINARY"
  else
    echo "sudo install failed, falling back to $HOME/.local/bin..." >&2
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
    mv "$DEST" "$INSTALL_DIR/$BINARY"
  fi
else
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
  mv "$DEST" "$INSTALL_DIR/$BINARY"
fi

chmod +x "$INSTALL_DIR/$BINARY" 2>/dev/null || true

echo "✓ flotio installed to $INSTALL_DIR/$BINARY"

# --- PATH check ---
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo ""
    echo "⚠ $INSTALL_DIR is not in your PATH."
    echo "  Add this to your ~/.bashrc or ~/.zshrc:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    ;;
esac
