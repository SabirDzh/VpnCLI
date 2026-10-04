#!/bin/sh
# Install vpn CLI (+ sing-box core) from GitHub releases.
# Usage: curl -fsSL https://raw.githubusercontent.com/SabirDzh/VpnCLI/master/install.sh | sh
set -eu

REPO="SabirDzh/VpnCLI"
BIN="vpn"
MIN_SINGBOX="${MIN_SINGBOX:-1.14.0}"
SKIP_SINGBOX="${SKIP_SINGBOX:-0}"

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

# ver_ge A B: true if A >= B (sort -V comparison).
ver_ge() {
  [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]
}

singbox_version() {
  sing-box version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n1 || true
}

ensure_singbox() {
  if [ "$SKIP_SINGBOX" = "1" ]; then
    echo "skipping sing-box setup (SKIP_SINGBOX=1)"
    return
  fi
  if have sing-box; then
    ver="$(singbox_version)"
    if [ -n "$ver" ] && ver_ge "$ver" "$MIN_SINGBOX"; then
      echo "sing-box $ver already installed (>= $MIN_SINGBOX)"
      return
    fi
    echo "sing-box $ver is too old (need >= $MIN_SINGBOX), upgrading..."
  else
    echo "installing sing-box core (>= $MIN_SINGBOX)..."
  fi
  case "$OS" in
    darwin)
      have brew || { echo "need Homebrew to install sing-box: https://brew.sh" >&2; exit 1; }
      # install exits 0 even when the keg is present but unlinked,
      # so always (re)link afterwards; both are safe no-ops otherwise.
      HOMEBREW_NO_AUTO_UPDATE=1 brew install sing-box || true
      HOMEBREW_NO_AUTO_UPDATE=1 brew link --overwrite sing-box 2>/dev/null || true
      ;;
    linux)
      if have apt-get; then
        curl -fsSL https://sing-box.app/gpg.key | sudo gpg --dearmor -o /usr/share/keyrings/sagernet-keyring.gpg
        echo 'deb [signed-by=/usr/share/keyrings/sagernet-keyring.gpg] https://deb.sagernet.org/ * *' \
          | sudo tee /etc/apt/sources.list.d/sagernet.list > /dev/null
        sudo apt-get update && sudo apt-get install -y sing-box
      else
        echo "no apt-get: install sing-box >= $MIN_SINGBOX manually:" >&2
        echo "https://sing-box.sagernet.org/installation/package-manager/" >&2
        exit 1
      fi
      ;;
  esac
  ver="$(singbox_version)"
  if [ -z "$ver" ] || ! ver_ge "$ver" "$MIN_SINGBOX"; then
    echo "sing-box install failed or too old: ${ver:-missing}" >&2
    exit 1
  fi
  echo "sing-box $ver ready"
}

install_vpn() {
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
}

ensure_singbox
install_vpn
