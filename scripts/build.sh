#!/usr/bin/env bash
# FYRwall release build script.
# Builds the frontend (bun) and the Go binary with version metadata,
# placing the binary in dist/.
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-$(cat VERSION 2>/dev/null || echo 0.1.0)}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo none)"
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
GO=go
BUN=bun

echo "[1/3] Building frontend"
if command -v "$BUN" >/dev/null 2>&1; then
  (cd web && "$BUN" install --frozen-lockfile >/dev/null 2>&1 || "$BUN" install >/dev/null)
  (cd web && "$BUN" run build)
else
  echo "bun not found; trying npm/npx"
  (cd web && npm install --no-audit --no-fund >/dev/null)
  (cd web && npm run build)
fi

echo "[2/3] Syncing embedded assets"
rm -rf webembed/dist
mkdir -p webembed/dist
cp -r web/dist/* webembed/dist/

echo "[3/3] Building fyrwall ($VERSION, $COMMIT)"
mkdir -p dist
$GO build -trimpath -ldflags "-s -w \
  -X github.com/PotenFYR-Studios/FYRwall/internal/version.Version=$VERSION \
  -X github.com/PotenFYR-Studios/FYRwall/internal/version.Commit=$COMMIT \
  -X github.com/PotenFYR-Studios/FYRwall/internal/version.BuildDate=$DATE" \
  -o dist/fyrwall ./cmd/fyrwall

echo "Built dist/fyrwall"
./dist/fyrwall version
