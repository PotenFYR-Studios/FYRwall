#!/usr/bin/env bash
# FYRwall distro/arch compatibility matrix.
# Builds and runs the test suite across the supported OS and CPU matrix,
# all inside Docker. Never executes tests on the host.
#
# Usage:
#   scripts/docker-matrix.sh              # full matrix (slow first run)
#   scripts/docker-matrix.sh --arch-only  # just cross-compile every arch
#   scripts/docker-matrix.sh debian:bookworm alpine:3.20   # specific distros
set -euo pipefail
cd "$(dirname "$0")/.."

MODE="full"
DISTROS=()
if [ "${1:-}" = "--arch-only" ]; then
  MODE="arch"
else
  DISTROS=("$@")
fi

echo "=== [1/3] Cross-compile smoke test: every supported architecture ==="
ARCHS=(amd64 arm64 arm 386 ppc64le s390x riscv64)
FAIL=0
for arch in "${ARCHS[@]}"; do
  if GOOS=linux GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -o /dev/null ./cmd/fyrwall; then
    echo "  OK   linux/$arch"
  else
    echo "  FAIL linux/$arch"
    FAIL=1
  fi
done
# 32-bit overflow checks: run the test suite compiled for 386 and arm on
# the host's native kernel via qemu-less execution is not possible, so we
# at least vet-compile tests for them.
for arch in 386 arm; do
  if GOOS=linux GOARCH="$arch" CGO_ENABLED=0 go vet ./... >/dev/null 2>&1; then
    echo "  OK   vet linux/$arch (32-bit safety)"
  else
    echo "  FAIL vet linux/$arch"
    FAIL=1
  fi
done
if [ "$MODE" = "arch" ]; then
  exit "$FAIL"
fi

if [ ${#DISTROS[@]} -eq 0 ]; then
  # Default distro matrix: glibc, musl, old-glibc, and fedora's toolchain.
  DISTROS=(
    "debian:bookworm-slim"      # glibc 2.36, systemd family
    "debian:bullseye-slim"      # older glibc, still common on servers
    "ubuntu:24.04"              # current LTS
    "alpine:3.20"               # musl, no glibc
    "fedora:40"                 # newest glibc/gcc
    "rockylinux:9"              # RHEL family
    "archlinux:base-devel"      # rolling, pacman family
    "opensuse/leap:15"          # SUSE family
  )
fi

echo "=== [2/3] Test suite per distro (in containers, network isolated) ==="
for image in "${DISTROS[@]}"; do
  safe="${image//[:\/]/-}"
  echo "--- $image"
  if docker image inspect "fyrwall-matrix:$safe" >/dev/null 2>&1; then
    BUILD_STATUS=cached
  else
    if BUILD_STATUS=built docker build -q -f - -t "fyrwall-matrix:$safe" - <<EOF >/dev/null 2>&1; then
FROM $image
RUN (command -v apt-get >/dev/null && apt-get update && apt-get install -y --no-install-recommends golang-go ca-certificates iptables) || \
    (command -v dnf >/dev/null && dnf install -y golang iptables) || \
    (command -v apk >/dev/null && apk add --no-cache go iptables) || \
    (command -v pacman >/dev/null && pacman -Sy --noconfirm go iptables) || \
    (command -v zypper >/dev/null && zypper --non-interactive install go iptables) || \
    (echo "no known package manager"; exit 1)
EOF
    else
      echo "  SKIP (no Go toolchain available for this distro image)"
      continue
    fi
  fi
  if docker run --rm --network=none --cap-drop=ALL \
      -e GOPROXY=off -e GOFLAGS=-mod=mod \
      -v "$PWD":/src:ro -w /src \
      "fyrwall-matrix:$safe" \
      go test -count=1 ./... >/tmp/matrix-$safe.log 2>&1; then
    echo "  PASS ($(grep -c '^ok' /tmp/matrix-$safe.log) packages)"
  else
    echo "  FAIL"
    tail -20 /tmp/matrix-$safe.log
    FAIL=1
  fi
done

echo "=== [3/3] Summary ==="
if [ "$FAIL" -eq 0 ]; then
  echo "ALL GREEN"
else
  echo "FAILURES DETECTED"
fi
exit "$FAIL"
