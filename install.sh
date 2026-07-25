#!/usr/bin/env bash
# Install loopbudget-claude-code (static binary). No Node required.
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/LoopBudget/cli/main/install.sh | VERSION=0.1.0 bash
set -euo pipefail

REPO="${LOOPBUDGET_REPO:-LoopBudget/cli}"
VERSION="${VERSION:-0.1.0}"
PREFIX="${PREFIX:-$HOME/.loopbudget}"
BIN_DIR="$PREFIX/bin"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "Unsupported arch: $arch" >&2; exit 1 ;;
esac
case "$os" in
  darwin|linux) ;;
  *) echo "Unsupported OS: $os (Windows: download .exe from GitHub Releases)" >&2; exit 1 ;;
esac

asset="loopbudget-claude-code_${VERSION}_${os}_${arch}"
base="https://github.com/${REPO}/releases/download/v${VERSION}"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

echo "Downloading $asset …"
curl -fsSL "$base/$asset" -o "$tmpdir/$asset"
curl -fsSL "$base/SHA256SUMS" -o "$tmpdir/SHA256SUMS"

(
  cd "$tmpdir"
  if command -v shasum >/dev/null 2>&1; then
    grep " $asset\$" SHA256SUMS | shasum -a 256 -c -
  else
    grep " $asset\$" SHA256SUMS | sha256sum -c -
  fi
)

mkdir -p "$BIN_DIR"
install -m 755 "$tmpdir/$asset" "$BIN_DIR/loopbudget-claude-code"

echo "Installed $BIN_DIR/loopbudget-claude-code"
if ! command -v loopbudget-claude-code >/dev/null 2>&1; then
  echo "Add to PATH, e.g.:"
  echo "  export PATH=\"$BIN_DIR:\$PATH\""
fi
echo "Next: loopbudget-claude-code init && loopbudget-claude-code doctor"
