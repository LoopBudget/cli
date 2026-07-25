#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${VERSION:-0.2.0}"
OUT="${OUT:-$ROOT/dist}"
mkdir -p "$OUT"

bins=(loopbudget-claude-code loopbudget-cursor)
targets=(
  "darwin arm64"
  "darwin amd64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
)

cd "$ROOT"
for bin in "${bins[@]}"; do
  for pair in "${targets[@]}"; do
    set -- $pair
    GOOS=$1 GOARCH=$2
    ext=""
    [[ "$GOOS" == "windows" ]] && ext=".exe"
    name="${bin}_${VERSION}_${GOOS}_${GOARCH}${ext}"
    echo "→ $name"
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o "$OUT/$name" "./cmd/$bin"
  done
done

cd "$OUT"
shasum -a 256 loopbudget-*_${VERSION}_* > SHA256SUMS
echo "Artifacts in $OUT"
cat SHA256SUMS
