#!/bin/sh
# Install vpn CLI from GitHub releases.
# Usage: curl -fsSL https://raw.githubusercontent.com/SabirDzh/VpnCLI/master/install.sh | sh
set -eu

REPO="SabirDzh/VpnCLI"
BIN="vpn"

os="$(uname -s)"
arch="$(uname -m)"
case "$os" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac
case "$arch" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

have() { command -v "$1" >/dev/null 2>&1; }
if have curl; then
  fetch() { curl -fsSL "$1" -o "$2"; }
elif have wget; then
  fetch() { wget -qO "$2" "$1"; }
else
  echo "need curl or wget" >&2; exit 1
fi

VERSION="${VERSION:-latest}"
if [ "$VERSION" = "latest" ]; then
  TAG="$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest" | sed 's/.*\///')"
  [ -n "$TAG" ] || { echo "cannot resolve latest release" >&2; exit 1; }
else
  TAG="$VERSION"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM
VER="${TAG#v}" # release assets use the version without leading v
echo "installing $BIN $TAG ($OS/$ARCH)..."
fetch "https://github.com/$REPO/releases/download/$TAG/${BIN}_${VER}_${OS}_${ARCH}.tar.gz" "$tmp/vpn.tar.gz"
tar -xzf "$tmp/vpn.tar.gz" -C "$tmp"

DEST="${PREFIX:-/usr/local}/bin/$BIN"
DESTDIR="$(dirname "$DEST")"
[ -d "$DESTDIR" ] || mkdir -p "$DESTDIR" 2>/dev/null || sudo mkdir -p "$DESTDIR"
if [ -w "$DESTDIR" ]; then
  install -m 0755 "$tmp/$BIN" "$DEST"
else
  echo "need sudo for $DESTDIR, retrying with sudo..."
  sudo install -m 0755 "$tmp/$BIN" "$DEST"
fi
echo "installed to $DEST"
"$DEST" version
