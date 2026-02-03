#!/bin/bash
# SAME installer — downloads the appropriate binary for your platform.
# Usage: curl -fsSL https://raw.githubusercontent.com/sgx-labs/statelessagent/main/install.sh | bash

set -euo pipefail

REPO="sgx-labs/statelessagent"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Detect platform
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Darwin)
    case "$ARCH" in
      arm64) SUFFIX="darwin-arm64" ;;
      x86_64) SUFFIX="darwin-amd64" ;;
      *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    ;;
  Linux)
    case "$ARCH" in
      x86_64) SUFFIX="linux-amd64" ;;
      *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    ;;
  MINGW*|MSYS*|CYGWIN*|Windows*)
    SUFFIX="windows-amd64.exe"
    ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

# Get latest release tag
echo "Fetching latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "No releases found. Building from source..."
  if command -v go >/dev/null 2>&1; then
    CGO_ENABLED=1 go build -ldflags "-s -w" -o "$INSTALL_DIR/same" ./cmd/same
    echo "Built $INSTALL_DIR/same from source"
    exit 0
  fi
  echo "Go not found. Install Go or wait for a release."
  exit 1
fi

BINARY_NAME="same-$SUFFIX"
URL="https://github.com/$REPO/releases/download/$LATEST/$BINARY_NAME"

echo "Downloading SAME $LATEST for $SUFFIX..."
mkdir -p "$INSTALL_DIR"

OUTPUT="$INSTALL_DIR/same"
if [[ "$SUFFIX" == *".exe" ]]; then
  OUTPUT="$INSTALL_DIR/same.exe"
fi

curl -fsSL "$URL" -o "$OUTPUT"
chmod +x "$OUTPUT"

echo ""
echo "Installed: $OUTPUT"
echo "Version: $("$OUTPUT" version 2>/dev/null || echo "$LATEST")"
echo ""
echo "Next steps:"
echo "  1. Run: $OUTPUT reindex"
echo "  2. Hooks are configured in .claude/settings.json"
echo "  3. MCP server: $OUTPUT mcp"
