#!/usr/bin/env bash
# FYRwall installer.
#
# Review-first installation is recommended:
#   curl -fsSLo install-fyrwall.sh https://fyrwall.docs.potenfyr.in/install.sh
#   less install-fyrwall.sh
#   sudo sh install-fyrwall.sh
#
# The installer never modifies firewall rules. It creates the service
# account and directories, installs binaries and units, and runs
# non-destructive preflight checks at the end.
#
# Binary resolution order:
#   1. a release tarball extracted next to the script or in CWD
#      (fyrwall_<version>_linux_<arch>/) - air-gapped installs
#   2. dist/fyrwall (local build output)
#   3. the official GitHub release for FYRWALL_VERSION (or the latest
#      release), checksum-verified against the release SHA256SUMS.
# Set FYRWALL_NO_DOWNLOAD=1 to forbid step 3, or FYRWALL_VERSION=vX.Y.Z
# (plain "X.Y.Z" also works) to pin a version.
# POSIX mode on purpose: the documented invocation is "sudo sh install.sh",
# and dash (Debian/Ubuntu /bin/sh) rejects "set -o pipefail".
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "error: run the installer as root (sudo sh install.sh)" >&2
  exit 1
fi

if [ "$(uname -s)" != "Linux" ]; then
  echo "error: FYRwall is Linux-only (detected $(uname -s))" >&2
  exit 1
fi

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  GOARCH=amd64 ;;
  aarch64|arm64) GOARCH=arm64 ;;
  armv7l|armv6l) GOARCH=arm ;;
  i686|i386) GOARCH=386 ;;
  ppc64le) GOARCH=ppc64le ;;
  s390x)   GOARCH=s390x ;;
  riscv64) GOARCH=riscv64 ;;
  *) echo "error: unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

VERSION="${FYRWALL_VERSION:-$(cat VERSION 2>/dev/null || echo 0.1.0)}"
VERSION="${VERSION#v}"
PREFIX="${FYRWALL_PREFIX:-/usr/local}"
if [ "$PREFIX" = "/usr" ]; then BINDIR=/usr/bin; else BINDIR="$PREFIX/bin"; fi

fetch() {
  # fetch <url> <outfile>; returns non-zero when no fetcher exists
  if command -v curl >/dev/null 2>&1; then
    curl -fsSLo "$2" "$1"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$2" "$1"
  else
    return 127
  fi
}

echo "[1/7] Creating service account"
if ! id fyrwall >/dev/null 2>&1; then
  useradd --system --home-dir /var/lib/fyrwall --shell /usr/sbin/nologin fyrwall
fi

echo "[2/7] Creating directories"
install -d -m 0750 -o fyrwall -g fyrwall /var/lib/fyrwall
install -d -m 0750 -o root  -g fyrwall /var/lib/fyrwall/backups
install -d -m 0750 -o root  -g fyrwall /var/lib/fyrwall/restore-points
install -d -m 0750 -o fyrwall -g fyrwall /var/log/fyrwall
install -d -m 0750 -o root  -g fyrwall /run/fyrwall
install -d -m 0750 -o root  -g fyrwall /etc/fyrwall

echo "[3/7] Installing binary"
# 1. Local tarball extract (air-gapped installs) or 2. dist/ build output.
BIN=""
if [ -f "fyrwall_${VERSION}_linux_${GOARCH}/fyrwall" ]; then
  BIN="fyrwall_${VERSION}_linux_${GOARCH}/fyrwall"
elif [ -f "dist/fyrwall" ]; then
  BIN="dist/fyrwall"
fi

# 3. Download the official release and verify its checksum.
if [ -z "$BIN" ]; then
  if [ -n "${FYRWALL_NO_DOWNLOAD:-}" ]; then
    echo "error: no fyrwall binary found; extract a release tarball next to the installer, build with scripts/build.sh, or unset FYRWALL_NO_DOWNLOAD" >&2
    exit 1
  fi
  if [ -z "${FYRWALL_VERSION:-}" ]; then
    echo "      resolving the latest release"
    LATEST="$(fetch https://api.github.com/repos/PotenFYR-Studios/FYRwall/releases/latest /tmp/fyrwall-release.json 2>/dev/null && sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' /tmp/fyrwall-release.json | head -n 1 || true)"
    rm -f /tmp/fyrwall-release.json
    if [ -n "$LATEST" ]; then
      VERSION="${LATEST#v}"
      echo "      latest release: $VERSION"
    else
      echo "error: could not resolve the latest release (offline?); pin one with FYRWALL_VERSION=X.Y.Z" >&2
      exit 1
    fi
  fi
  TMP="$(mktemp -d)"
  TARBALL="fyrwall_${VERSION}_linux_${GOARCH}.tar.gz"
  BASE="https://github.com/PotenFYR-Studios/FYRwall/releases/download/v${VERSION}"
  echo "      downloading $TARBALL"
  if ! fetch "$BASE/$TARBALL" "$TMP/$TARBALL"; then
    echo "error: download failed (offline host? use FYRWALL_NO_DOWNLOAD with a local tarball)" >&2
    rm -rf "$TMP"
    exit 1
  fi
  if fetch "$BASE/SHA256SUMS" "$TMP/SHA256SUMS" && command -v sha256sum >/dev/null 2>&1; then
    grep "linux_${GOARCH}.tar.gz" "$TMP/SHA256SUMS" | (cd "$TMP" && sha256sum -c -) || {
      echo "error: checksum verification failed for $TARBALL" >&2
      rm -rf "$TMP"
      exit 1
    }
    echo "      checksum verified"
  else
    echo "error: could not fetch SHA256SUMS or sha256sum is missing; refusing an unverified install" >&2
    rm -rf "$TMP"
    exit 1
  fi
  tar -xzf "$TMP/$TARBALL" -C "$TMP"
  BIN="$TMP/fyrwall_${VERSION}_linux_${GOARCH}/fyrwall"
  BIN_TMP="$TMP"
fi

install -m 0755 "$BIN" "$BINDIR/fyrwall"
if [ -n "${BIN_TMP:-}" ]; then rm -rf "$BIN_TMP"; fi

echo "[4/7] Installing config (existing config is never overwritten)"
if [ ! -f /etc/fyrwall/config.yaml ]; then
  if [ -f "fyrwall_${VERSION}_linux_${GOARCH}/config.example.yaml" ]; then
    install -m 0640 -o root -g fyrwall "fyrwall_${VERSION}_linux_${GOARCH}/config.example.yaml" /etc/fyrwall/config.yaml
  elif [ -f packaging/config.example.yaml ]; then
    install -m 0640 -o root -g fyrwall packaging/config.example.yaml /etc/fyrwall/config.yaml
  fi
fi

echo "[4b/7] Registering as installed application (desktop entry)"
install -d -m 0755 /usr/share/applications /usr/share/icons/hicolor/scalable/apps
install -m 0644 packaging/applications/fyrwall.desktop /usr/share/applications/ 2>/dev/null || \
  echo "  (desktop entry skipped - packaging/applications missing)"
[ -f packaging/fyrwall-icon.svg ] && install -m 0644 packaging/fyrwall-icon.svg /usr/share/icons/hicolor/scalable/apps/fyrwall.svg
update-desktop-database /usr/share/applications 2>/dev/null || true

echo "[5/7] Installing systemd units"
if command -v systemctl >/dev/null 2>&1; then
  if [ -d "fyrwall_${VERSION}_linux_${GOARCH}/packaging/systemd" ]; then
    install -m 0644 fyrwall_${VERSION}_linux_${GOARCH}/packaging/systemd/*.service /etc/systemd/system/
  elif [ -d packaging/systemd ]; then
    install -m 0644 packaging/systemd/*.service /etc/systemd/system/
  fi
  systemctl daemon-reload
else
  echo "  systemd not detected; enable FYRwall manually with your init system"
fi

echo "[6/7] Verifying checksums"
if [ -f dist/SHA256SUMS ] && command -v sha256sum >/dev/null 2>&1; then
  grep "linux_${GOARCH}" dist/SHA256SUMS | (cd dist && sha256sum -c -) || {
    echo "error: checksum verification failed" >&2; exit 1;
  }
fi

echo "[7/7] Running non-destructive preflight"
"$BINDIR/fyrwall" preflight --config /etc/fyrwall/config.yaml || {
  echo "warning: preflight reported issues; FYRwall will start in DEGRADED/BLOCKED state with details in the UI"
}

cat <<EOF

Installation complete.

  Binary:   $BINDIR/fyrwall
  Config:   /etc/fyrwall/config.yaml
  Data:     /var/lib/fyrwall
  Version:  $VERSION

Next steps:
  1. Set an admin password and bootstrap:
       FYRWALL_ADMIN_PASSWORD='choose-a-long-random-password' sudo -E fyrwall user create-admin
  2. Start services (optional, only if you want them now):
       systemctl enable --now fyrwall-agent fyrwall-server
  3. Open http://127.0.0.1:7443 and log in.

This installer never touched your firewall rules.
EOF
