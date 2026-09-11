#!/usr/bin/env bash
# FYRwall dockerized test runner.
# ALL Go tests run inside the container; nothing executes on the host.
# Usage:
#   scripts/docker-test.sh              # full suite
#   scripts/docker-test.sh ./internal/firewall/...   # targeted packages
set -euo pipefail
cd "$(dirname "$0")/.."

IMAGE=fyrwall-test:latest

echo "[1/2] Building test image (cached after first run)"
docker build -f Dockerfile.test -t "$IMAGE" . >/dev/null

echo "[2/2] Running tests in container"
PACKAGES="${1:-./...}"
docker run --rm \
  --network=none \
  --read-only \
  --tmpfs /tmp:rw,exec,size=1024m \
  --cap-drop=ALL \
  --security-opt no-new-privileges \
  -e GOPROXY=off \
  -e GOCACHE=/tmp/go-cache \
  -v "$PWD":/src:ro \
  "$IMAGE" \
  go test -count=1 -v "$PACKAGES"
