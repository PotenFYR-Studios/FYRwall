# Changelog

All notable changes to FYRwall are documented here. Format follows
Keep a Changelog; versions follow Semantic Versioning.

## [0.1.0] - 2026-09-11

### Added
- Core firewall management: UFW and iptables (legacy and nft flavor)
  adapters with normalized rule model, validation and conflict detection
- Firewall ownership detection with write blocking on
  MULTIPLE_CONFLICTING owners
- Transactional applies: snapshot, apply, verify, automatic rollback,
  rollback-failure escalation to critical notification
- Argon2id authentication, server-side sessions, CSRF, RBAC with five
  roles, per-IP login rate limiting, session revocation
- SQLite persistence (pure-Go driver) with ordered migrations; audit
  log; deduplicated persistent notifications
- Startup state machine with preflight, 16 read-only diagnostics checks,
  health aggregation (STARTING, HEALTHY, DEGRADED, BLOCKED)
- Typed Unix-socket agent (0660) with strictly allowlisted operations
- React + TypeScript web UI embedded in the Go binary; no Node.js on
  production hosts
- Extension system: declarative, capability-scoped manifests for
  dashboard widgets, notification channels, diagnostics probes, rule
  templates and sandboxed UI panels; execution-class capabilities can
  never be granted
- Optional version-update notification (off by default) with a
  comparison engine and updates manifest format
- Secure, retention-bounded logging for server and agent: 0640 files,
  size-based rotation, age-based deletion, optional AES-256-GCM
  encrypted agent logs with a host-local key
- Docker-only test harness plus distro/arch compatibility matrix
  (8 distros, 7 architectures)
- install.sh / uninstall.sh with clean removal and optional config
  retention; hardened systemd units for server and agent
