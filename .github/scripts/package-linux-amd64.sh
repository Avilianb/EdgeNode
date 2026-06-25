#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
VERSION="$(sed -nE 's/^[[:space:]]*Version[[:space:]]*=[[:space:]]*"([0-9.]+)".*/\1/p' "$ROOT/internal/const/const.go" | head -n 1)"

if [[ -z "$VERSION" ]]; then
  echo "could not detect edge-node version" >&2
  exit 1
fi

rm -rf "$ROOT/dist"

DIST="$ROOT/dist/edge-node"
mkdir -p "$DIST/bin" "$DIST/configs" "$DIST/logs" "$DIST/data"

cp "$ROOT/build/configs/api_node.template.yaml" "$DIST/configs/"
cp "$ROOT/build/configs/cluster.template.yaml" "$DIST/configs/"
cp -R "$ROOT/build/www" "$DIST/"
cp -R "$ROOT/build/pages" "$DIST/"

if [[ -f "$ROOT/build/edge-toa/edge-toa-amd64" ]]; then
  mkdir -p "$DIST/edge-toa"
  cp "$ROOT/build/edge-toa/edge-toa-amd64" "$DIST/edge-toa/edge-toa"
fi

GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
  go build -trimpath -tags community -ldflags="-s -w" \
  -o "$DIST/bin/edge-node" \
  "$ROOT/cmd/edge-node/main.go"

find "$DIST" -name ".DS_Store" -delete
find "$DIST" -name ".gitignore" -delete

(
  cd "$ROOT/dist"
  zip -r -X -q "edge-node-linux-amd64-community-v${VERSION}.zip" edge-node/
)

ASSET="$ROOT/dist/edge-node-linux-amd64-community-v${VERSION}.zip"
if [[ ! -s "$ASSET" ]]; then
  echo "missing release asset: $ASSET" >&2
  exit 1
fi

(
  cd "$ROOT/dist"
  sha256sum "$(basename "$ASSET")" > SHA256SUMS.txt
)
