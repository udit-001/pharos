#!/bin/sh
set -eu

REPO="udit-001/pharos"
BIN="pharos"
VERSION="${1:-latest}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)   OS="linux" ;;
  darwin)  OS="darwin" ;;
  mingw*|msys*|cygwin*) OS="windows" ;;
  *)       echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect arch
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)            echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

# Resolve version to tag
if [ "$VERSION" = "latest" ]; then
  TAG=$(curl -sfL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
  [ -z "$TAG" ] && { echo "Failed to fetch latest release"; exit 1; }
else
  TAG="$VERSION"
fi

# Strip leading 'v' for filename
VER="${TAG#v}"
ARCHIVE="pharos_${VER}_${OS}_${ARCH}.tar.gz"
[ "$OS" = "windows" ] && ARCHIVE="pharos_${VER}_${OS}_${ARCH}.zip"

echo "Downloading $BIN $TAG for $OS/$ARCH..."

URL="https://github.com/$REPO/releases/download/$TAG/$ARCHIVE"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -sfL "$URL" -o "$TMP/$ARCHIVE"

if [ "$OS" = "windows" ]; then
  unzip -o "$TMP/$ARCHIVE" -d "$TMP" >/dev/null 2>&1
else
  tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
fi

# Find binary (GoReleaser places it at root of archive)
BIN_PATH="$TMP/$BIN"
[ ! -f "$BIN_PATH" ] && [ -f "$TMP/${BIN}.exe" ] && BIN_PATH="$TMP/${BIN}.exe"
[ ! -f "$BIN_PATH" ] && BIN_PATH=$(find "$TMP" -type f \( -name "$BIN" -o -name "${BIN}.exe" \) | head -1)
[ -z "$BIN_PATH" ] && { echo "Binary not found in archive"; exit 1; }

chmod +x "$BIN_PATH"

# Install
DEST="/usr/local/bin"
if [ ! -w "$DEST" ]; then
  DEST="${HOME}/.local/bin"
  mkdir -p "$DEST"
fi

mv "$BIN_PATH" "$DEST/$BIN"
echo "Installed $BIN $TAG to $DEST/$BIN"

# Warn if the install destination isn't on PATH (common for ~/.local/bin)
case ":$PATH:" in
  *":$DEST:"*) ;;
  *) echo "Note: $DEST is not on your PATH. Add it:"
     echo "  export PATH=\"$DEST:\$PATH\"" ;;
esac
