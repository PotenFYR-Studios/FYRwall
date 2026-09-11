#!/usr/bin/env bash
# FYRwall cross-compile release script.
# Produces tarballs for the supported architecture matrix in dist/,
# plus a SHA256SUMS manifest.
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-$(cat VERSION 2>/dev/null || echo 0.1.0)}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo none)"
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
LDFLAGS="-s -w -X github.com/PotenFYR-Studios/FYRwall/internal/version.Version=$VERSION -X github.com/PotenFYR-Studios/FYRwall/internal/version.Commit=$COMMIT -X github.com/PotenFYR-Studios/FYRwall/internal/version.BuildDate=$DATE"

# Ensure embedded assets exist.
if [ ! -f webembed/dist/index.html ]; then
  echo "webembed/dist is empty; run scripts/build.sh first (or a full frontend build)." >&2
  exit 1
fi

OUT=dist
mkdir -p "$OUT"

PLATFORMS=(
  "linux amd64"
  "linux arm64"
  "linux arm"
  "linux 386"
  "linux ppc64le"
  "linux s390x"
  "linux riscv64"
)

for entry in "${PLATFORMS[@]}"; do
  read -r GOOS GOARCH <<< "$entry"
  NAME="fyrwall_${VERSION}_${GOOS}_${GOARCH}"
  WORK="$OUT/$NAME"
  echo "Building $NAME"
  mkdir -p "$WORK"
  GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 \
    go build -trimpath -ldflags "$LDFLAGS" -o "$WORK/fyrwall" ./cmd/fyrwall
  # Asset and service files.
  cp README.md LICENSE "$WORK/" 2>/dev/null || true
  mkdir -p "$WORK/packaging/systemd"
  cp packaging/systemd/*.service "$WORK/packaging/systemd/" 2>/dev/null || true
  cp packaging/config.example.yaml "$WORK/" 2>/dev/null || true
  tar -czf "$OUT/$NAME.tar.gz" -C "$OUT" "$NAME"
  rm -rf "$WORK"
done

echo "Generating checksums"
( cd "$OUT" && for f in ./*.tar.gz; do shasum -a 256 "$f"; done > SHA256SUMS )

echo "Release artifacts in $OUT:"
ls -1 "$OUT"
