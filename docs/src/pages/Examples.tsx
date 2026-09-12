// Examples: every snippet here is lifted from the repository itself
// (README, packaging/config.example.yaml, docker-compose.yml,
// docs/content). No invented flags or endpoints.
import { Link } from "react-router-dom";
import { CopyButton } from "../components/buttons";

function Snippet({
  title,
  caption,
  lang,
  code,
  link,
  linkLabel,
}: {
  title: string;
  caption: string;
  lang?: string;
  code: string;
  link?: string;
  linkLabel?: string;
}) {
  return (
    <section className="mt-10">
      <h2 className="text-[1.15em] font-bold text-white">{title}</h2>
      <p className="mt-1 text-[0.9em] text-muted">{caption}</p>
      <pre
        className="mt-3 overflow-x-auto rounded-[10px] border border-line bg-panel p-4 font-mono text-[0.84em] leading-[1.65] text-[#dfe2ef] shadow-[var(--shadow)]"
        data-lang={lang ?? ""}
      >
        <code>{code}</code>
        <CopyButton text={code} />
      </pre>
      {link && (
        <p className="mt-2 text-[0.85em]">
          <Link to={link}>{linkLabel ?? "Full guide"}</Link>
        </p>
      )}
    </section>
  );
}

const CLI = `# status and health
fyrwall status                # firewall + ownership summary
fyrwall doctor                # read-only diagnostics
fyrwall preflight             # diagnostics + backend detection
fyrwall service status        # show detected init system

# configuration
fyrwall config validate --config /etc/fyrwall/config.yaml
fyrwall db migrate            # apply pending migrations

# first admin (password via environment, never argv)
FYRWALL_ADMIN_PASSWORD='a-long-random-password' fyrwall user create-admin

# restore points
fyrwall restore create        # manual restore point
fyrwall restore list          # list restore points

# extensions
sudo fyrwall extension install ./my-extension   # prompts per capability
sudo fyrwall extension list

# updates (optional, off by default)
fyrwall update check          # compare installed vs latest manifest
fyrwall update apply          # backup + restore point + verified download

# desktop and removal
fyrwall tray                  # system tray icon (Open Web UI, Restart, Stop, Quit)
fyrwall uninstall             # asks what to keep; never touches firewall rules`;

const CONFIG = `server:
  bind: "127.0.0.1"      # non-loopback without TLS is refused
  port: 7443
tls:
  enabled: false
database:
  driver: "sqlite"       # sqlite | postgres (with postgres_dsn)
  sqlite_path: "/var/lib/fyrwall/fyrwall.db"
firewall:
  backend: "auto"                        # auto | ufw | iptables
  safe_apply_timeout_seconds: 60
  allow_write_on_manager_conflict: false # explicit acknowledgment only
logging:
  level: "info"
  format: "json"
  file_path: "/var/log/fyrwall/fyrwall.log"
restore_points:
  auto_enabled: true
  retain_automatic: 50
  retain_manual: 20
security:
  session_idle_timeout_minutes: 30
  login_rate_limit_per_minute: 5`;

const ENV = `FYRWALL_SERVER_BIND=127.0.0.1
FYRWALL_SERVER_PORT=7443
FYRWALL_DB_SQLITE_PATH=/var/lib/fyrwall/fyrwall.db
FYRWALL_LOG_LEVEL=info
FYRWALL_FIREWALL_BACKEND=auto
FYRWALL_TLS_ENABLED=false
FYRWALL_TLS_CERT=/path/to/cert.pem
FYRWALL_TLS_KEY=/path/to/key.pem
FYRWALL_SAFE_APPLY_TIMEOUT=60
FYRWALL_ALLOW_INSECURE_BIND=false     # explicit opt-in for non-TLS non-loopback binds
FYRWALL_ADMIN_PASSWORD=...            # create-admin reads the password here`;

const FIRST_BOOT = `# 1. install (never touches your firewall rules)
curl -fsSL https://fyrwall.docs.potenfyr.in/install.sh | sudo sh

# 2. bootstrap the first admin - password from the environment
FYRWALL_ADMIN_PASSWORD='choose-a-long-random-password' sudo -E fyrwall user create-admin

# 3. start services
sudo systemctl enable --now fyrwall-agent fyrwall-server

# 4. open the UI and log in
#    http://127.0.0.1:7443`;

const DOCKER = `# server only (UI + API)
docker run -d --name fyrwall \\
  -p 127.0.0.1:7443:7443 \\
  -v fyrwall-data:/var/lib/fyrwall \\
  ghcr.io/potenfyr-studios/fyrwall:latest

# server + bridged host agent (recommended)
docker compose up -d`;

const EXTENSION = `# my-extension/extension.yaml
id: my-battery-widget
name: UPS Battery Widget
version: 1.0.0
author: you
min_app: 0.1.0
capabilities:
  - dashboard.widget          # every entry needs an explicit admin grant
widgets:
  - id: ups-charge
    title: UPS charge
    kind: metric              # metric | gauge | table | status
    size: small
    source: http.local
    query: {"url": "http://127.0.0.1:8080/charge"}
    refresh_seconds: 30

# install: prints the capability list and asks for grants
sudo fyrwall extension install /path/to/my-battery-widget`;

const RULE_TEMPLATE = `# rules.template packs use the normalized rule model
[{"direction": "in",
  "action": "allow",
  "protocol": "tcp",
  "destination_port": "443"}]`;

const API = `# public
GET  /api/v1/version
GET  /api/v1/system/health
POST /api/v1/auth/login          POST /api/v1/auth/logout      GET /api/v1/auth/me

# authenticated (session + CSRF)
GET  /api/v1/firewall/status     GET  /api/v1/firewall/rules
POST /api/v1/firewall/rules/validate
POST /api/v1/firewall/transactions
POST /api/v1/firewall/snapshots
GET  /api/v1/settings            PUT  /api/v1/settings
GET  /api/v1/users               POST /api/v1/users
GET  /api/v1/system/diagnostics
GET  /api/v1/audit               GET  /api/v1/notifications

# JSON envelope with machine-readable error codes:
# FW_BACKEND_CONFLICT, FW_VALIDATION_FAILED, AUTH_RATE_LIMITED, ...`;

export default function Examples() {
  return (
    <div className="dot-backdrop">
      <div className="mx-auto max-w-4xl px-6 pb-20 pt-14">
        <p className="eyebrow">
          <span className="eyebrow-accent">FYRwall</span> examples
        </p>
        <h1 className="grad-text mt-3 text-[clamp(2.2em,5vw,3.2em)] font-extrabold leading-[1.08] tracking-tight">
          Copy, paste, ship
        </h1>
        <p className="mt-4 max-w-2xl text-[1.04em] text-muted">
          Working snippets from the repository: install, first boot,
          configuration, Docker, the CLI, extension manifests and the API.
          Everything on this page runs as shown on a supported Linux host.
        </p>

        <Snippet
          title="Install"
          caption="The one-liner, or review-first if you prefer reading the script before running it."
          lang="shell"
          code={`curl -fsSL https://fyrwall.docs.potenfyr.in/install.sh | sudo sh

# review-first
curl -fsSLo install-fyrwall.sh https://fyrwall.docs.potenfyr.in/install.sh
less install-fyrwall.sh
sudo sh install-fyrwall.sh`}
          link="/docs/installation"
          linkLabel="Installation guide"
        />

        <Snippet
          title="First boot"
          caption="Bootstrap the admin, start the services, log in."
          lang="shell"
          code={FIRST_BOOT}
          link="/docs/operation"
          linkLabel="Operation guide"
        />

        <Snippet
          title="Minimal config"
          caption="Every key is optional; values shown are the defaults. Place at /etc/fyrwall/config.yaml."
          lang="yaml"
          code={CONFIG}
          link="/docs/configuration"
          linkLabel="Configuration reference"
        />

        <Snippet
          title="Environment overrides"
          caption="FYRWALL_* environment variables override the config file."
          lang="ini"
          code={ENV}
          link="/docs/configuration"
          linkLabel="Configuration reference"
        />

        <Snippet
          title="Docker"
          caption="Server-only container, or the compose file with a host-networked agent bridge."
          lang="shell"
          code={DOCKER}
          link="/docs/docker"
          linkLabel="Docker guide"
        />

        <Snippet
          title="CLI cookbook"
          caption="Every command below ships in the binary today - verified against the cobra command tree."
          lang="shell"
          code={CLI}
        />

        <Snippet
          title="Extension manifest"
          caption="Declarative, capability-scoped. Denied by default; the installer asks per capability."
          lang="yaml"
          code={EXTENSION}
          link="/docs/extensions-guide"
          linkLabel="Extensions guide"
        />

        <Snippet
          title="Rule template pack"
          caption="Templates carry rules in the normalized JSON model the validators check."
          lang="json"
          code={RULE_TEMPLATE}
          link="/docs/extensions-guide"
          linkLabel="Extensions guide"
        />

        <Snippet
          title="API surface"
          caption="All endpoints are versioned under /api/v1 with a JSON envelope."
          lang="http"
          code={API}
        />
      </div>
    </div>
  );
}
