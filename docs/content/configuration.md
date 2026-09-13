# Configuration

FYRwall reads `/etc/fyrwall/config.yaml`. Every key is optional with a
sensible default, and `FYRWALL_*` environment variables override the file.
Validate any config before restarting:

    fyrwall config validate --config /etc/fyrwall/config.yaml

## Server

    server:
      bind: "127.0.0.1"      # loopback by default; non-loopback without TLS is refused
      port: 7443
      public_url: ""
      trusted_proxies: []    # only list proxies you control

- `bind`: loopback by default. A non-loopback bind without TLS is refused
  unless `FYRWALL_ALLOW_INSECURE_BIND=true`.
- `public_url`: your external URL, for links and agent enrollment.
- `trusted_proxies`: reverse proxies allowed to set client IP headers.

## TLS

    tls:
      enabled: false
      cert_file: ""
      key_file: ""

TLS 1.2+ when enabled. Security headers are always on regardless.

## Database

    database:
      driver: "sqlite"       # sqlite | postgres
      sqlite_path: "/var/lib/fyrwall/fyrwall.db"
      postgres_dsn: ""

SQLite (pure-Go driver, WAL, single-writer pooling) is the storage engine.
The config schema also accepts `driver: "postgres"` (with a required
`postgres_dsn`), reserved for an upcoming backend; SQLite remains the
supported default.

## Firewall

    firewall:
      backend: "auto"                        # auto | ufw | iptables
      safe_apply_timeout_seconds: 60
      allow_write_on_manager_conflict: false

- `backend`: `auto` detects UFW or iptables; explicit `ufw` or `iptables`
  pins an adapter.
- `safe_apply_timeout_seconds`: seconds before a connectivity-breaking
  change rolls back automatically.
- `allow_write_on_manager_conflict`: explicit acknowledgment only.

## Agent and fleet

    agent:
      server_url: ""
      enrollment_token: ""
      credential_file: "/var/lib/fyrwall/agent-credential.json"
      id_file: "/var/lib/fyrwall/agent-id"
      display_name: ""
      labels: []
      groups: []
      site: ""

- `server_url`: central HTTPS URL for outbound fleet synchronization.
- `enrollment_token`: single-use first-boot token. Prefer the
  `FYRWALL_AGENT_ENROLLMENT_TOKEN` environment variable so it is not stored.
- `credential_file` and `id_file`: root-owned 0600 agent identity state.
- `labels`, `groups`, and `site`: fleet targeting and inventory metadata.

## Service

    service:
      autostart: true

## Logging

    logging:
      level: "info"          # debug | info | warn | error | fatal
      format: "json"         # json | console
      file_enabled: true
      file_path: "/var/log/fyrwall/fyrwall.log"

Server logs are 0640, size- and age-bounded, with secret redaction. Agent
logs are AES-256-GCM encrypted at rest with a host-local 0600 key.

## Restore points

    restore_points:
      auto_enabled: true
      retain_automatic: 50
      retain_manual: 20
      retention_days: 365

Automatic restore points before every change; manual points via
`fyrwall restore create`.

## Security

    security:
      session_idle_timeout_minutes: 30
      login_rate_limit_per_minute: 5

Argon2id hashing, secure cookies, CSRF tokens, per-IP rate limiting and
RBAC are always on; these keys tune the session and login throttle.

## Environment overrides

    FYRWALL_SERVER_BIND=127.0.0.1
    FYRWALL_SERVER_PORT=7443
    FYRWALL_DB_SQLITE_PATH=/var/lib/fyrwall/fyrwall.db
    FYRWALL_LOG_LEVEL=info
    FYRWALL_FIREWALL_BACKEND=auto
    FYRWALL_TLS_ENABLED=false
    FYRWALL_TLS_CERT=/path/to/cert.pem
    FYRWALL_TLS_KEY=/path/to/key.pem
    FYRWALL_SAFE_APPLY_TIMEOUT=60
    FYRWALL_ALLOW_INSECURE_BIND=false
    FYRWALL_AGENT_SERVER_URL=https://fw.example.com
    FYRWALL_AGENT_ENROLLMENT_TOKEN=<single-use-token>

Passwords and tokens arrive via the environment or secret-file references,
never via command-line arguments. The initial super admin password is
chosen in the web setup wizard on first boot, not provisioned via env.

## Encrypted config at rest

Config is stored encrypted (FYRCFG1 format) with AES-256-GCM and a
root-owned 0600 keyfile, with tamper detection and a one-time plaintext
migration. Writes go through the authenticated web UI only.

Next: [Operation](operation.md) · [Safety and Restore Points](safety.md)
