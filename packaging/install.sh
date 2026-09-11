#!/usr/bin/env bash
# FYRwall installer.
#
# Review-first installation is recommended:
#   curl -fsSLo install-fyrwall.sh <url>/install.sh
#   less install-fyrwall.sh
#   sudo sh install-fyrwall.sh
#
# The installer never modifies firewall rules. It creates the service
# account and directories, installs binaries and units, and runs
# non-destructive preflight checks at the end.
set -euo pipefail

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

VERSION="${FYRWALL_VERSION:-0.1.0}"
PREFIX="${FYRWALL_PREFIX:-/usr/local}"
if [ "$PREFIX" = "/usr" ]; then BINDIR=/usr/bin; else BINDIR="$PREFIX/bin"; fi

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
# Prefer a local tarball if provided (air-gapped installs), else dist/ output.
if [ -f "fyrwall_${VERSION}_linux_${GOARCH}/fyrwall" ]; then
  install -m 0755 "fyrwall_${VERSION}_linux_${GOARCH}/fyrwall" "$BINDIR/fyrwall"
elif [ -f "dist/fyrwall" ]; then
  install -m 0755 dist/fyrwall "$BINDIR/fyrwall"
else
  echo "error: no fyrwall binary found; build with scripts/build.sh or extract a release tarball first" >&2
  exit 1
fi

echo "[4/7] Installing config (existing config is never overwritten)"
if [ ! -f /etc/fyrwall/config.yaml ]; then
  install -m 0640 -o root -g fyrwall packaging/config.example.yaml /etc/fyrwall/config.yaml
fi

echo "[4b/7] Registering as installed application (desktop entry)"
install -d -m 0755 /usr/share/applications /usr/share/icons/hicolor/scalable/apps
install -m 0644 packaging/applications/fyrwall.desktop /usr/share/applications/ 2>/dev/null || \
  echo "  (desktop entry skipped - packaging/applications missing)"
[ -f packaging/fyrwall-icon.svg ] && install -m 0644 packaging/fyrwall-icon.svg /usr/share/icons/hicolor/scalable/apps/fyrwall.svg
update-desktop-database /usr/share/applications 2>/dev/null || true

echo "[5/7] Installing systemd units"
if command -v systemctl >/dev/null 2>&1; then
  install -m 0644 packaging/systemd/fyrwall-server.service /etc/systemd/system/
  install -m 0644 packaging/systemd/fyrwall-agent.service  /etc/systemd/system/
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

Next steps:
  1. Set an admin password and bootstrap:
       FYRWALL_ADMIN_PASSWORD='choose-a-long-random-password' sudo -E fyrwall user create-admin
  2. Start services (optional, only if you want them now):
       systemctl enable --now fyrwall-agent fyrwall-server
  3. Open http://127.0.0.1:7443 and log in.

This installer never touched your firewall rules.
EOF
