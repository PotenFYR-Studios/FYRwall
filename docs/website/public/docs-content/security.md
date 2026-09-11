# Security Model

## Privilege separation

- fyrwall-server: unprivileged. Never runs firewall commands.
- fyrwall-agent: unprivileged. Collects state and events.
- Privileged helper: tiny root daemon, Unix socket 0660, strictly
  allowlisted typed operations, revalidates every request, SO_PEERCRED
  peer checks. No generic exec API exists anywhere.

## Authentication

Argon2id password hashing (constant-time verify), server-side sessions
with secure cookies, CSRF tokens on all mutations, per-IP login rate
limiting, session revocation on password change.

## Transport

TLS 1.2+ when enabled; security headers always (CSP frame-ancestors
none, nosniff, no-referrer). Agent traffic is mutually authenticated.
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
