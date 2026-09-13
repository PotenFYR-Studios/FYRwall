<div align="center">

<img src="https://capsule-render.vercel.app/api?type=waving&color=0:8b5cf6,50:ec4899,100:f97316&height=220&section=header&text=FYRwall&fontSize=52&fontColor=ffffff&fontAlignY=34&animation=twinkling" width="100%" alt="FYRwall banner"/>

[![Typing SVG](https://readme-typing-svg.demolab.com?font=Fira+Code&weight=600&size=20&pause=1200&color=8B5CF6&center=true&vCenter=true&width=800&lines=Transactional+firewall+changes+with+auto-rollback;Lockout+protection+built+in+%F0%9F%9B%A1%EF%B8%8F;UFW+%7C+iptables+%7C+nftables+-+one+GUI;Unprivileged+by+design.+No+shell.+No+root+UI.)](https://github.com/PotenFYR-Studios/FYRwall)

[![CI](https://img.shields.io/github/actions/workflow/status/PotenFYR-Studios/FYRwall/ci.yml?style=for-the-badge&logo=githubactions&logoColor=white&label=CI&labelColor=1c1e26&color=2ea043)](https://github.com/PotenFYR-Studios/FYRwall/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/PotenFYR-Studios/FYRwall?style=for-the-badge&logo=github&logoColor=white&label=Release&labelColor=1c1e26&color=eac54f)](https://github.com/PotenFYR-Studios/FYRwall/releases)
[![Docker](https://img.shields.io/badge/ghcr.io-fyrwall-2496ED?style=for-the-badge&logo=docker&logoColor=white&labelColor=1c1e26)](https://github.com/PotenFYR-Studios/FYRwall/pkgs/container/fyrwall)
[![Docs](https://img.shields.io/badge/Docs-fyrwall.docs.potenfyr.in-8b5cf6?style=for-the-badge&logo=readme&logoColor=white&labelColor=1c1e26)](https://fyrwall.docs.potenfyr.in/)
[![License](https://img.shields.io/badge/License-Apache--2.0%20%2B%20Commons%20Clause-2ea043?style=for-the-badge&logo=apache&logoColor=white&labelColor=1c1e26)](https://github.com/PotenFYR-Studios/FYRwall/blob/master/LICENSE)
[![View](https://komarev.com/ghpvc/?username=PotenFYR-Studios-FYRwall&color=ec4899&style=for-the-badge&label=VIEW&labelColor=1c1e26)](https://github.com/PotenFYR-Studios/FYRwall)

[Overview](#overview) · [Install](#install) · [Docker](#docker) · [Docs](https://fyrwall.docs.potenfyr.in/) · [Extensions](#extensions) · [FAQ](#faq) · [Releases](https://github.com/PotenFYR-Studios/FYRwall/releases)

```bash
curl -fsSL https://fyrwall.docs.potenfyr.in/install.sh | sudo sh
```

</div>

---

## Overview

A safe, focused, modern web-based firewall administration platform for Linux. FYRwall gives administrators a clean GUI over UFW and iptables without ever running the web server as root, without shell interpolation anywhere, and without a single firewall mutation that lacks a restore point and rollback path.

Built by **PotenFYR Studios**.

---

## Core Guarantees

- The web/API server runs unprivileged. Only the tiny local agent touches the firewall, through a strictly permissioned Unix socket.
- Every privileged operation is a strongly typed, allowlisted request. There is no generic exec endpoint and no shell string anywhere in the codebase.
- Every firewall mutation follows the full pipeline: authorize, lock, validate, detect conflicts, snapshot, apply, re-read, verify, and automatic rollback on any failure after the snapshot step.
- Firewall ownership detection (UFW, iptables-legacy, iptables-nft, firewalld, native nftables) blocks writes whenever multiple independent managers are active. Nothing is ever disabled silently.
- Failed or suspicious changes roll back automatically. Rollback failure raises a critical notification, never a silent success.

---

## Features

### Firewall Management
- Normalized rule model across backends: family, direction, action, protocol, source/destination, ports, interfaces, conntrack state, comments, priority
- Read current status, default policies, and full rule lists
- Create, validate, and apply rule transactions with structural validation (CIDR, ports, family mismatches, interface names, loopback safety)
- Conflict engine: duplicate rules, shadowed rules, allow/deny overlap, SSH lockout risk, management-port lockout, with severity levels and remediation hints
- UFW adapter: parses `status verbose` and `status numbered` including v6 sections, LIMIT rules, multiport specs, and parenthesized comments
- iptables adapter: iptables-save parser handling multiport, conntrack, quoted comments; legacy vs nft flavor detection; iptables-restore for rollback
- Ownership/conflict detectors for firewalld and native nftables rulesets

### Safety Machinery
- Snapshot/restore engine with SHA-256 state hashes
- Automatic pre-change restore points on every transaction
- Verify-after-apply with hash comparison and automatic rollback on mismatch
- Critical conflicts (SSH lockout, loopback denial) hard-block applies at the validation layer

### Authentication and Access
- Argon2id password hashing (PHC format, constant-time verification, rehash detection)
- Server-side sessions with secure cookie defaults, idle timeout, and token renewal on login (fixation defense)
- CSRF protection on all state-changing methods
- Five roles: super admin, admin, operator, auditor, viewer; permissions enforced in backend middleware, not just the UI
- Per-IP login rate limiting with bounded memory
- Admin-driven session revocation

### Operations
- Startup state machine: STARTING, HEALTHY, DEGRADED, BLOCKED, STOPPING
- 16 read-only diagnostic checks (kernel, netfilter, binaries, IPv6, init system, disk, clock, loopback, route, DNS, and more)
- Persistent, deduplicated notifications: same issue recurring increments an occurrence counter instead of spamming
- Startup issues automatically surface as notifications, log events, and dashboard entries
- Structured JSON logging with secret redaction, output size caps on every subprocess
- SQLite storage via the pure-Go modernc driver (WAL, foreign keys, integrity checks), ordered migrations covering users, audit, notifications, rules, restore points, templates, and fleet tables

### Web UI
- React 18 + TypeScript + Vite + Tailwind CSS
- Dashboard with firewall status cards, policy owner badge, conflict banner for multi-manager hosts
- Rules table with live refresh
- Health page with per-component status
- Notification center with severity badges
- Embedded into the Go binary at build time; the production host needs no Node.js

### Agent
- Narrow privileged Unix-socket service at `/run/fyrwall/agent.sock` (0660); web server remains unprivileged
- Strictly allowlisted typed operations; unknown operations are rejected and logged
- Per-connection deadlines and bounded request sizes
- Single-use remote enrollment tokens and hashed per-agent credentials
- mTLS client certificates with automatic rotation and per-agent revocation
- Outbound restart-safe long polling; no inbound management ports on targets
- Ordered state revisions, drift hashes, durable commands, result retry, and reconnect reconciliation
- Fleet inventory, labels, groups, site metadata, health, backend ownership, and policy revisions
- Staged group rollouts with configurable batch size and failure budget
- Remote command progress plus per-target and group emergency rollback

---

## Security Model

| Layer | Control |
|---|---|
| Process separation | Unprivileged server; typed Unix-socket agent; no root web server |
| Command execution | Argument arrays only, resolved binary paths, minimal fixed env, output caps, timeouts |
| Input handling | Strict JSON decoding (unknown fields and trailing data rejected), request body size limits |
| Auth | Argon2id, secure cookies, CSRF tokens, login rate limiting, session fixation defense |
| Authorization | RBAC middleware on every route, enforced in services |
| Transport headers | CSP with frame-ancestors none, nosniff, X-Frame-Options DENY, no-referrer, HSTS when TLS is on |
| Firewall safety | Ownership gating, restore points, verify, rollback, rollback-failure escalation |
| Secrets | Redacted from logs and diagnostics; passwords accepted via environment variables only, never argv |
| Config at rest | AES-256-GCM encrypted config (FYRCFG1 format) with a root-owned 0600 keyfile; 0640 config, tamper detection via GCM auth, automatic one-time plaintext migration; GUI-only write path |

---

## Repository Layout

```
cmd/fyrwall/            CLI entrypoint (cobra)
internal/
  app/                  Lifecycle wiring, startup, preflight, admin bootstrap
  api/                  chi router, middleware, handlers, strict JSON, security headers
  auth/                 Argon2id, sessions, CSRF, RBAC, rate limiting
  agent/                Unix-socket agent server with allowlisted ops
  config/               YAML + FYRWALL_* env config with validation
  database/             SQLite, migrations, user/audit/notification repos
  diagnostics/          Read-only check registry
  firewall/             Backend interface, models, validation, conflicts, ownership, manager
    ufw/                UFW adapter and parsers
    iptables/           iptables adapter, save-format parser
    detectors/          firewalld / nftables / ufw-service detection
  health/               State machine and aggregation
  logging/              Structured logging with redaction
  system/               Platform detection; exec (argument-array process runner)
  version/              Build info
web/                    React/TS frontend source (built with bun)
webembed/               Embedded frontend dist served by the Go binary
scripts/                docker-test.sh, build.sh, cross-build.sh
packaging/              systemd units, install.sh
test/                   Fixtures
```

---

## Building

Requirements: Go 1.24+, bun (or Node 20+) for the frontend, GNU make or plain shell.

### 1. Frontend

```bash
cd web
bun install
bun run build          # outputs web/dist
```

### 2. Sync embedded assets and build the binary

```bash
rm -rf webembed/dist && mkdir -p webembed/dist
cp -r web/dist/* webembed/dist/
go build -trimpath -ldflags "-s -w \
  -X github.com/PotenFYR-Studios/FYRwall/internal/version.Version=$(cat VERSION 2>/dev/null || echo 0.1.0) \
  -X github.com/PotenFYR-Studios/FYRwall/internal/version.Commit=$(git rev-parse --short HEAD 2>/dev/null || echo none) \
  -X github.com/PotenFYR-Studios/FYRwall/internal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o fyrwall ./cmd/fyrwall
```

Or simply:

```bash
./scripts/build.sh           # builds frontend + binary into dist/
./scripts/cross-build.sh     # cross-compiles the supported matrix into dist/
```

### 3. Release builds

Cross-compiled artifacts land in `dist/`:

```
fyrwall_<version>_linux_amd64.tar.gz
fyrwall_<version>_linux_arm64.tar.gz
fyrwall_<version>_linux_arm.tar.gz
fyrwall_<version>_linux_386.tar.gz
fyrwall_<version>_linux_ppc64le.tar.gz
fyrwall_<version>_linux_s390x.tar.gz
fyrwall_<version>_linux_riscv64.tar.gz
```

Each tarball contains the `fyrwall` binary, default config, systemd units, LICENSE, and README. SHA256SUMS is generated alongside.

---

## Testing

All Go tests run inside Docker, never on the host:

```bash
./scripts/docker-test.sh                        # full suite
./scripts/docker-test.sh ./internal/firewall/... # targeted packages
```

The runner builds `Dockerfile.test` (golang:1.24-bookworm + iptables), then executes with `--network=none --read-only --cap-drop=ALL` and offline module resolution. Current status: 46 tests passing across 6 packages.

Frontend:

```bash
cd web && bun install && bun run build   # typecheck (tsc) + vite build
```

---

## Running

### Quick start (local)

```bash
# config with a writable sqlite path
FYRWALL_DB_SQLITE_PATH=./fyrwall.db ./fyrwall server --config config.yaml

./fyrwall doctor                 # read-only diagnostics
./fyrwall preflight              # diagnostics + backend detection
./fyrwall status                 # firewall + ownership summary
```

The UI listens on `127.0.0.1:7443` by default. A non-loopback bind without TLS is refused unless `FYRWALL_ALLOW_INSECURE_BIND=true`.

### First boot

1. The server runs preflight, detects the platform and firewall backend, opens the database, and applies migrations.
2. Ownership is resolved. Multiple active managers puts FYRwall in DEGRADED with writes blocked and a persistent critical notification.
3. Open `http://127.0.0.1:7443`: on a fresh install the one-time setup wizard
   creates the super admin account and its password right in the browser
   (works the same for Docker - there is no CLI password step).
4. Sign in; the UI issues a CSRF token automatically.

### Configuration

`/etc/fyrwall/config.yaml` (all keys optional, sensible defaults):

```yaml
server:
  bind: "127.0.0.1"
  port: 7443
tls:
  enabled: false
  cert_file: ""
  key_file: ""
database:
  driver: "sqlite"
  sqlite_path: "/var/lib/fyrwall/fyrwall.db"
firewall:
  backend: "auto"                 # auto | ufw | iptables
  safe_apply_timeout_seconds: 60
  allow_write_on_manager_conflict: false
service:
  autostart: true
logging:
  level: "info"
  format: "json"
  file_enabled: true
  file_path: "/var/log/fyrwall/fyrwall.log"
restore_points:
  auto_enabled: true
  retain_automatic: 50
  retain_manual: 20
security:
  session_idle_timeout_minutes: 30
  login_rate_limit_per_minute: 5
```

Environment overrides: `FYRWALL_SERVER_BIND`, `FYRWALL_SERVER_PORT`, `FYRWALL_DB_SQLITE_PATH`, `FYRWALL_LOG_LEVEL`, `FYRWALL_FIREWALL_BACKEND`, `FYRWALL_TLS_ENABLED`, `FYRWALL_TLS_CERT`, `FYRWALL_TLS_KEY`, `FYRWALL_SAFE_APPLY_TIMEOUT`, `FYRWALL_ALLOW_INSECURE_BIND`.

Validate with `./fyrwall config validate --config your.yaml`.

---

## API Overview

All endpoints are versioned under `/api/v1`. JSON envelope with machine-readable error codes (`FW_BACKEND_CONFLICT`, `FW_VALIDATION_FAILED`, `AUTH_RATE_LIMITED`, and friends).

Public: `GET /version`, `GET /system/health`, `GET /setup/status`, `POST /setup/super-admin` (first run only), `POST /auth/login`, `POST /auth/logout`, `GET /auth/me`

Authenticated (session + CSRF required):
- `GET /firewall/status`, `GET /firewall/rules`
- `POST /firewall/rules/validate`, `POST /firewall/transactions`, `POST /firewall/snapshots`
- `GET /settings`, `PUT /settings`
- `GET /users`, `POST /users`
- `GET /system/diagnostics`
- `GET /audit`, `GET /notifications`
- `GET /agents`, `POST /agents/enrollment-tokens`
- `GET /agents/{id}/firewall/status`, `GET /agents/{id}/firewall/rules`
- `POST /agents/{id}/firewall/transactions`, `POST /agents/{id}/commands`

RBAC is enforced per route: viewers read, operators apply, admins manage users and settings, auditors get the audit trail.

---

## CLI Reference

```
fyrwall server              run the web/API server (unprivileged)
fyrwall agent               run the local firewall agent (Unix socket)
fyrwall agent --server URL  enroll/synchronize with a central server
fyrwall status              firewall and ownership summary
fyrwall doctor              read-only diagnostics
fyrwall preflight           diagnostics + backend detection
fyrwall config validate     validate a config file
fyrwall db migrate          apply pending migrations
fyrwall restore list        list restore points
fyrwall restore create      create a manual restore point
fyrwall service status      show detected init system
fyrwall version             print build info
```

---

## Docker

Run the server and preview the intended host-agent topology:

```bash
# Server only (UI + API)
docker run -d --name fyrwall \
  -p 127.0.0.1:7443:7443 \
  -v fyrwall-data:/var/lib/fyrwall \
  ghcr.io/potenfyr-studios/fyrwall:latest

# Server + co-located host agent topology
docker compose up -d
```

The server consumes the agent's typed Unix socket; only the sidecar gets the
host namespace and firewall capabilities. Remote agents dial out through
authenticated, restart-safe long-poll sessions, requiring no inbound management
port on targets. Full guide: [docs/content/docker.md](docs/content/docker.md).

## Updating

FYRwall tells you when a new release ships (opt-in, off by default - no telemetry):

```yaml
updates:
  check_enabled: true
```

A persistent notification with changelog link appears in the web UI. Update manually or semi-automatically:

```bash
fyrwall update check    # compare installed vs latest manifest
fyrwall update apply    # backup + restore point + verified download + migrate + restart
```

Every update creates a database backup and firewall restore point first; config is never overwritten. Re-running the curl one-liner is always a safe in-place upgrade. See [docs/content/updating.md](docs/content/updating.md).

## Auto build, releases and containers

| Event | What happens |
|---|---|
| push to `master` | CI test matrix + container image rebuild; existing release changelog gets a build line; rolling image tags refreshed |
| version bump tag `vX.Y.Z` | Full release: new GitHub Release with changelog + 7-arch tarballs + SHA256SUMS + new `:vX.Y.Z` image tag |
| push to `docs/**` | Docs site auto-deploys to GitHub Pages |

Same version, new commits = refreshed builds and changelog build-lines, no new release. Bumped version = new tag, new release, new image tag. Fully automatic.

## Architecture

```text
        Browser (React, embedded single binary)
                         |
                         v
  +--------------------------------------------------+
  |        fyrwall-server  (unprivileged)            |
  |  UI | API | RBAC | audit | notifications | PKI   |
  +------------------------+-------------------------+
                           | Unix socket 0660, typed ops only
                           v
  +--------------------------------------------------+
  |        fyrwall-agent (narrow root boundary)      |
  |    UFW adapter | iptables (legacy/nft) adapters  |
  |    ownership detection | conflict engine         |
  +------------------------+-------------------------+
                           |
                           v
             Linux netfilter (iptables/nftables)
```

Agents on managed targets dial out through authenticated, restart-safe sessions
for near-realtime state, drift detection, health, inventory, policy operations,
and centralized audit. A co-located agent makes the central server host another
managed target. Single-use enrollment tokens, durable command delivery,
sequence reconciliation, and persisted results handle reconnects safely.

## Extensions

Extend FYRwall without forking: dashboard widgets, notification channels (webhook/ntfy/Gotify/Discord), diagnostics probes (http/tcp/file), rule templates, event enrichers and sandboxed UI panels. Extensions are declarative manifests; every capability is denied by default and admin-granted at install. Execution, direct firewall mutation, user management and credential access are never grantable - those stay core-only by design.

```bash
sudo fyrwall extension install ./my-extension   # prompts per capability
sudo fyrwall extension list
```

Developer guide with full manifest reference: [docs/content/extensions-guide.md](docs/content/extensions-guide.md).

## Installed application: console command, tray and app menu

FYRwall installs as a real application. The installer drops the binary into `/usr/local/bin` (or `/usr/bin`), so from any console:

```bash
fyrwall            # same as before: server, agent, status, doctor, setup...
fyrwall tray       # desktop tray/console menu
```

Desktop Linux with a tray registers a status icon with: Open Web UI, Restart services, Stop services, Status, Quit. It also installs a `.desktop` entry, so FYRwall appears in your application menu (System > Security) and can be launched like any GUI app.

Uninstall with zero leftovers:

```bash
sudo sh packaging/uninstall.sh --all           # keeps config + restore points (asks)
sudo sh packaging/uninstall.sh --all --purge   # removes everything incl. service user
# or: fyrwall uninstall
```

FYRwall install and uninstall never modify your firewall rules.

## Testing and compatibility matrix

All tests run in Docker, never on the host:

```bash
./scripts/docker-test.sh        # unit + integration suite (46 tests, 6 packages)
./scripts/docker-matrix.sh      # 8 distros x 7 architectures compatibility matrix
```

Matrix covers Debian (bookworm, bullseye), Ubuntu, Alpine (musl), Fedora, Rocky, Arch and openSUSE on amd64, arm64, arm, 386, ppc64le, s390x and riscv64.

## Docs site

Full documentation lives at [fyrwall.docs.potenfyr.in](https://fyrwall.docs.potenfyr.in/) with installation, configuration, operation, security model, architecture, safety and restore points, extension development, updating and troubleshooting guides. Built with Vite + React + TypeScript via bun; every route is prerendered to real HTML with per-route titles, canonical URLs, Open Graph and JSON-LD structured data (sitemap, robots included). The github.io path keeps working as a redirect.

## ⭐ Star History

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=potenfyr-studios/authcore,potenfyr-studios/statfyr,potenfyr-studios/discord-botlists,potenfyr-studios/shell-eggs,potenfyr-studios/prog-language-eggs,potenfyr-studios/minecraft-eggs,potenfyr-studios/database-eggs,potenfyr-studios/apicordon,potenfyr-studios/ojaj,potenfyr-studios/fyrwall,potenfyr-studios/echoingdeaths&type=Date&theme=dark" />
  <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=potenfyr-studios/authcore,potenfyr-studios/statfyr,potenfyr-studios/discord-botlists,potenfyr-studios/shell-eggs,potenfyr-studios/prog-language-eggs,potenfyr-studios/minecraft-eggs,potenfyr-studios/database-eggs,potenfyr-studios/apicordon,potenfyr-studios/ojaj,potenfyr-studios/fyrwall,potenfyr-studios/echoingdeaths&type=Date" />
  <img alt="Star history chart for all PotenFYR Studios public repositories" src="https://api.star-history.com/svg?repos=potenfyr-studios/authcore,potenfyr-studios/statfyr,potenfyr-studios/discord-botlists,potenfyr-studios/shell-eggs,potenfyr-studios/prog-language-eggs,potenfyr-studios/minecraft-eggs,potenfyr-studios/database-eggs,potenfyr-studios/apicordon,potenfyr-studios/ojaj,potenfyr-studios/fyrwall,potenfyr-studios/echoingdeaths&type=Date" width="80%" />
</picture>

Every public PotenFYR Studios repository on one live chart, served by [star-history.com](https://star-history.com).

---

## Contributing

Contributions make the open-source community such an amazing place to learn, inspire and create. Any contributions you make are **greatly appreciated** - see [CONTRIBUTING.md](CONTRIBUTING.md) and the [good first issues](https://github.com/PotenFYR-Studios/FYRwall/labels/good%20first%20issue). Security concerns: please use [SECURITY.md](SECURITY.md) (private vulnerability reporting), not public issues.

<a href="https://github.com/PotenFYR-Studios/FYRwall/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=PotenFYR-Studios/FYRwall" alt="FYRwall contributors" />
</a>
<a href="https://github.com/PotenFYR-Studios/FYRwall/stargazers">
  <img src="https://img.shields.io/github/stars/PotenFYR-Studios/FYRwall?style=social&label=Stars" alt="Live star count" />
</a>
<a href="https://github.com/PotenFYR-Studios/FYRwall/network/members">
  <img src="https://img.shields.io/github/forks/PotenFYR-Studios/FYRwall?style=social&label=Forks" alt="Live fork count" />
</a>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/PotenFYR-Studios/FYRwall/output/github-snake-dark.svg" />
  <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/PotenFYR-Studios/FYRwall/output/github-snake.svg" />
  <img alt="Contribution snake animation" src="https://raw.githubusercontent.com/PotenFYR-Studios/FYRwall/output/github-snake.svg" width="100%" />
</picture>

---

## Platform Support

Capability-based detection rather than distro assumptions. Works wherever Go 1.24 runs and either UFW or iptables exists: Debian/Ubuntu, RHEL/Fedora/Rocky/Alma, SUSE, Arch, Alpine, Void, Gentoo. Init systems: systemd, OpenRC, runit, s6, SysV fallback. Release targets: amd64, arm64, arm, 386, ppc64le, s390x, riscv64.

FYRwall is fully offline-capable: no CDN assets, no telemetry, no external calls at runtime.

---

## License

FYRwall is licensed under **Apache-2.0 with Commons Clause**.

- Free to use, study, modify, self-host and redistribute
- Commercial use is welcome: embedding FYRwall as a feature inside a
  larger product or service is explicitly allowed
- What is NOT allowed: selling FYRwall itself - offering the software,
  or a service whose value derives entirely or substantially from it,
  as a paid product (this includes managed-hosting-of-FYRwall alone)

This matches the whole PotenFYR-Studios org: the tool stays open and
auditable, and nobody gets to resell it as-is. The [LICENSE](LICENSE) file
(https://github.com/PotenFYR-Studios/FYRwall/blob/master/LICENSE) is the
authoritative legal text; this section is a plain-English summary.

### Restrictions at a glance

| | Allowed | Not allowed |
|---|:---:|:---:|
| Personal / internal use | ✅ | |
| Self-hosting for your company | ✅ | |
| Modifying and redistributing (same license) | ✅ | |
| Embedding FYRwall as a feature of a larger paid product | ✅ | |
| Building paid services around FYRwall | ✅ | |
| Selling FYRwall itself (or a copy) for a fee | | ❌ |
| Offering paid managed hosting of FYRwall alone | | ❌ |
| Paid support/consulting whose value is FYRwall itself | | ❌ |
| Removing LICENSE / attribution notices | | ❌ |

"Sell" here follows the Commons Clause definition: charging for a
product or service whose value derives entirely or substantially from
FYRwall itself. If FYRwall is a minor feature of something bigger, you
are fine. Questions or a commercial exception: [contact the org](https://potenfyr.in/).

---

<div align="center">

## 🎯 The FYRwall Promise

✨ **Safe by default** (every change snapshotted, verified, rolled back) · ⚡ **Fast** (single Go binary, embedded UI) · 🛡️ **Hardened** (unprivileged, encrypted config, tamper checks) · 🤝 **Community-driven** (extensions, open source)

[![GitHub](https://img.shields.io/badge/GitHub-PotenFYR--Studios-181717?style=for-the-badge&logo=github&logoColor=white&labelColor=1c1e26)](https://github.com/PotenFYR-Studios)
[![Website](https://img.shields.io/badge/Website-potenfyr.in-8b5cf6?style=for-the-badge&logo=googlechrome&logoColor=white&labelColor=1c1e26)](https://potenfyr.in/)
[![Community](https://img.shields.io/badge/Community-Discord-5865F2?style=for-the-badge&logo=discord&logoColor=white&labelColor=1c1e26)](https://discord.com/invite/zUaN2FPBec)
[![Docs](https://img.shields.io/badge/Docs-fyrwall.docs.potenfyr.in-f97316?style=for-the-badge&logo=readme&logoColor=white&labelColor=1c1e26)](https://fyrwall.docs.potenfyr.in/)

<!-- markdownlint-disable -->
<div align="center">

<img src="https://capsule-render.vercel.app/api?type=waving&color=0:f97316,50:ec4899,100:8b5cf6&height=120&section=footer&text=Made%20with%20%E2%9D%A4%EF%B8%8F%20by%20PotenFYR%20Studios&fontSize=22&fontColor=ffffff&animation=twinkling" width="100%" alt="footer"/>

</div>
<!-- markdownlint-enable -->
