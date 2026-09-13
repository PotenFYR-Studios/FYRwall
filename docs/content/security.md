# Security Model

## Privilege separation

- fyrwall-server: unprivileged. Never runs firewall commands.
- fyrwall-agent: narrow root service with only NET_ADMIN/NET_RAW capabilities.
  Accepts typed allowlisted operations; no generic command or shell endpoint.
- Agent socket: Unix socket 0660 in a root-owned, service-group 0750 directory, strictly
  allowlisted typed operations. No generic exec API exists anywhere.

## Authentication

Argon2id password hashing (constant-time verify), server-side sessions
with secure cookies, CSRF tokens on all mutations, per-IP login rate
limiting, session revocation on password change.

## Transport

TLS 1.2+ when enabled; security headers always (CSP frame-ancestors
none, nosniff, no-referrer). Remote agent traffic uses mTLS client certificates
plus per-agent bearer credentials. Certificates rotate automatically and are
individually revocable; raw credentials stay on agents and server stores hashes.
Domain binding rejects foreign Host headers.

## Logs are protected

Server logs: 0640, service-group readable only. Agent logs: AES-256-GCM
encrypted at rest with a host-local 0600 key. Only the web admin can
read logs through the authenticated API. Retention bounds every log by
size (50 MiB x 5 backups x 30 days default) so disk can never fill.

## Secrets

Never logged, never in diagnostics bundles, never on argv. Passwords
and tokens travel via environment or secret files (env:, file: refs).

## What extensions can never do

No execution, no direct firewall mutation, no user management, no
credential or key access. Extensions are declarative and
capability-scoped, denied by default, granted by an admin at install.

Next: [Safety and Restore Points](safety.md) · [Extensions](extensions.md)
