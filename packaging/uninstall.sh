#!/usr/bin/env bash
# FYRwall uninstaller: removes server and/or agent completely, with an
# option to keep the config and restore points for future reinstall.
#
#   sudo sh uninstall.sh              # interactive
#   sudo sh uninstall.sh --all        # server + agent, keep data
#   sudo sh uninstall.sh --all --purge # remove everything including restore points
set -euo pipefail

[ "$(id -u)" -ne 0 ] && { echo "run as root (sudo)"; exit 1; }

TARGET=""
PURGE=0
KEEP_CONFIG=""
for arg in "$@"; do
  case "$arg" in
    --server) TARGET="server" ;;
    --agent)  TARGET="agent" ;;
    --all)    TARGET="all" ;;
    --purge)  PURGE=1 ;;
    *) echo "unknown option $arg"; exit 1 ;;
  esac
done

if [ -z "$TARGET" ]; then
  echo "FYRwall uninstaller"
  select choice in "Uninstall server only" "Uninstall agent only" "Uninstall everything" "Cancel"; do
    case "$REPLY" in
      1) TARGET=server; break ;;
      2) TARGET=agent;  break ;;
      3) TARGET=all;    break ;;
      *) exit 0 ;;
    esac
  done
fi

if [ -z "$KEEP_CONFIG" ] && [ "$PURGE" -eq 0 ]; then
  echo
  printf "Keep /etc/fyrwall config (and restore points) for future reinstall? [Y/n]: "
  read -r ans
  case "$ans" in
    n|N) PURGE=1 ;;
    *)   PURGE=0; KEEP_CONFIG=yes ;;
  esac
fi

stop_disable() {
  if command -v systemctl >/dev/null 2>&1; then
    systemctl stop "$1" 2>/dev/null || true
    systemctl disable "$1" 2>/dev/null || true
    rm -f "/etc/systemd/system/$1"
  fi
}

echo "[1/5] Stopping and removing services"
[ "$TARGET" = "server" ] || [ "$TARGET" = "all" ] && stop_disable fyrwall-server.service
[ "$TARGET" = "agent"  ] || [ "$TARGET" = "all" ] && stop_disable fyrwall-agent.service
if command -v systemctl >/dev/null 2>&1; then systemctl daemon-reload 2>/dev/null || true; fi

echo "[2/5] Removing binaries"
rm -f /usr/bin/fyrwall /usr/local/bin/fyrwall

echo "[3/5] Removing runtime state"
rm -rf /run/fyrwall

echo "[4/5] Removing data and logs"
if [ "$PURGE" -eq 1 ]; then
  rm -rf /var/lib/fyrwall /var/log/fyrwall /etc/fyrwall
  echo "  purged config, database, restore points and logs"
else
  rm -rf /var/log/fyrwall/*.log* 
  echo "  kept /etc/fyrwall (config) and /var/lib/fyrwall (restore points, database)"
  echo "  remove manually later if desired: rm -rf /var/lib/fyrwall /etc/fyrwall"
fi

echo "[5/5] Removing service account"
if [ "$PURGE" -eq 1 ]; then
  userdel fyrwall 2>/dev/null || true
  echo "  removed user fyrwall"
else
  echo "  kept user fyrwall (purge to remove)"
fi

echo
echo "FYRwall $TARGET uninstall complete. Your firewall rules were never modified by FYRwall and remain as they are."
