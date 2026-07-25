#!/usr/bin/env bash
# Install LoopBudget CLIs (static binaries). No Node required.
#   curl -fsSL https://raw.githubusercontent.com/LoopBudget/cli/main/install.sh | VERSION=0.2.0 bash
set -euo pipefail

REPO="${LOOPBUDGET_REPO:-LoopBudget/cli}"
VERSION="${VERSION:-0.2.0}"
PREFIX="${PREFIX:-$HOME/.loopbudget}"
BIN_DIR="$PREFIX/bin"
TOOLS="${TOOLS:-claude-code,cursor}" # comma list: claude-code | cursor | both

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "Unsupported arch: $arch" >&2; exit 1 ;;
esac
case "$os" in
  darwin|linux) ;;
  *) echo "Unsupported OS: $os" >&2; exit 1 ;;
esac

base="https://github.com/${REPO}/releases/download/v${VERSION}"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

echo "Fetching SHA256SUMS …"
curl -fsSL "$base/SHA256SUMS" -o "$tmpdir/SHA256SUMS"
mkdir -p "$BIN_DIR"

install_one() {
  local name="$1"
  local asset="loopbudget-${name}_${VERSION}_${os}_${arch}"
  echo "Downloading $asset …"
  curl -fsSL "$base/$asset" -o "$tmpdir/$asset"
  (
    cd "$tmpdir"
    if command -v shasum >/dev/null 2>&1; then
      grep " $asset\$" SHA256SUMS | shasum -a 256 -c -
    else
      grep " $asset\$" SHA256SUMS | sha256sum -c -
    fi
  )
  install -m 755 "$tmpdir/$asset" "$BIN_DIR/loopbudget-${name}"
  echo "Installed $BIN_DIR/loopbudget-${name}"
}

IFS=',' read -ra parts <<<"$TOOLS"
for t in "${parts[@]}"; do
  t="$(echo "$t" | tr -d '[:space:]')"
  case "$t" in
    claude-code|cursor) install_one "$t" ;;
    *) echo "Unknown tool: $t (use claude-code or cursor)" >&2; exit 1 ;;
  esac
done

if ! echo "$PATH" | tr ':' '\n' | grep -qx "$BIN_DIR"; then
  echo "Add to PATH:  export PATH=\"$BIN_DIR:\$PATH\""
fi
echo "Next: loopbudget-claude-code init && loopbudget-claude-code doctor"
echo "      loopbudget-cursor   # after credentials exist"
