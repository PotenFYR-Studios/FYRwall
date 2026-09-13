#!/usr/bin/env bash
# Host the FYRwall server locally from this checkout (dev/test harness).
#
# Builds the web GUI, syncs it into webembed/dist, builds the Go binary,
# then runs the server with an isolated config: loopback bind, sqlite and
# logs under tests/.tmp, nothing touched outside the repo.
#
# Usage:
#   tests/run-server.sh                 # build if stale, serve on :7443
#   tests/run-server.sh --port 8080     # serve on another port
#   tests/run-server.sh --no-build      # reuse the previously built binary
#   tests/run-server.sh --rebuild-frontend
#
# Env:
#   PORT                     default port (7443); --port wins
#
# Ctrl-C stops the server. State (db, logs, integrity record) lives in
# tests/.tmp/ and is gitignored; delete that dir for a factory reset.
set -euo pipefail
cd "$(dirname "$0")/.."

PORT="${PORT:-7443}"
BUILD_FRONTEND="auto" # auto | yes | no
NO_BUILD=0

while [ $# -gt 0 ]; do
  case "$1" in
    --port) PORT="$2"; shift 2 ;;
    --no-build) NO_BUILD=1; BUILD_FRONTEND=no; shift ;;
    --rebuild-frontend) BUILD_FRONTEND=yes; shift ;;
    -h|--help) sed -n '2,22p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown option: $1 (see --help)" >&2; exit 2 ;;
  esac
done

WORKDIR=tests/.tmp
mkdir -p "$WORKDIR/config" "$WORKDIR/state" "$WORKDIR/logs" "$WORKDIR/bin"
CFG="$WORKDIR/config/config.yaml"
BIN="$WORKDIR/bin/fyrwall"

# ---- [1/4] Web GUI ----
frontend_stale() {
  [ ! -f web/dist/index.html ] && return 0
  [ -n "$(find web/src web/index.html web/tailwind.config.js web/postcss.config.js \
      web/vite.config.ts -newer web/dist/index.html -print -quit 2>/dev/null)" ]
}

if [ "$BUILD_FRONTEND" = auto ] && frontend_stale; then
  BUILD_FRONTEND=yes
fi

if [ "$BUILD_FRONTEND" = yes ]; then
  echo "[1/4] Building web GUI"
  if [ ! -d web/node_modules ]; then
    if command -v bun >/dev/null 2>&1; then (cd web && bun install)
    else (cd web && npm install --no-audit --no-fund); fi
  fi
  if command -v bun >/dev/null 2>&1; then (cd web && bun run build)
  else (cd web && npm run build); fi
else
  echo "[1/4] Web GUI up to date (web/dist)"
fi

# Sync embedded assets: a Go build against an empty webembed/dist ships a
# blank GUI (this exact bug), so always refresh before compiling.
rm -rf webembed/dist
mkdir -p webembed/dist
cp -r web/dist/* webembed/dist/
touch webembed/dist/.gitkeep # keep the tracked placeholder in place

# ---- [2/4] Go binary ----
if [ "$NO_BUILD" = 1 ] && [ -x "$BIN" ]; then
  echo "[2/4] Reusing $BIN"
else
  echo "[2/4] Building fyrwall"
  go build -o "$BIN" ./cmd/fyrwall
  # New binary hash: drop the stale integrity record so first boot re-records.
  rm -f "$WORKDIR/config/integrity.json"
fi

# ---- [3/4] Isolated config ----
cat > "$CFG" <<EOF
server:
  bind: "127.0.0.1"
  port: ${PORT}
tls:
  enabled: false
database:
  driver: "sqlite"
  sqlite_path: "$(pwd)/$WORKDIR/state/fyrwall.db"
firewall:
  backend: "auto"
logging:
  level: "info"
  format: "console"
  file_enabled: true
  file_path: "$(pwd)/$WORKDIR/logs/fyrwall.log"
EOF

if command -v ss >/dev/null 2>&1; then
  if ss -ltnH "sport = :$PORT" 2>/dev/null | grep -q .; then
    echo "port $PORT already in use; pick another with --port" >&2
    exit 1
  fi
fi

# ---- [4/4] Serve (super admin is set up through the web GUI on first run) ----
echo "[3/4] Config: $CFG"

echo
echo "  Web UI:  http://127.0.0.1:${PORT}"
echo "  First run: the GUI opens the one-time super admin setup wizard;"
echo "           choose the admin username and password there."
echo

echo "Serving on http://127.0.0.1:${PORT}  (logs: $WORKDIR/logs/fyrwall.log; Ctrl-C to stop)"
"$BIN" server --config "$CFG" &
SRV_PID=$!
trap 'kill "$SRV_PID" 2>/dev/null || true' EXIT INT TERM

for _ in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:${PORT}/api/v1/system/health" >/dev/null 2>&1; then
    echo "Ready: http://127.0.0.1:${PORT}"
    break
  fi
  if ! kill -0 "$SRV_PID" 2>/dev/null; then
    echo "server exited during startup; see $WORKDIR/logs/fyrwall.log" >&2
    exit 1
  fi
  sleep 0.5
done

wait "$SRV_PID"
