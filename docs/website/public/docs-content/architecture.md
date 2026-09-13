# Architecture

This is the implemented control-plane architecture.

    Browser (React, embedded)
        |
        v
    fyrwall-server (unprivileged control plane)
        |
        +-- local Unix socket --> co-located agent (optional)
        |
        +-- outbound authenticated sessions from remote agents
                                  |
                                  v
                         UFW / iptables / nftables

The agent belongs on every machine whose firewall FYRwall manages. An agent
may be explicitly installed beside the server so the central host becomes
another managed target. The server uses the local typed socket; remote agents
use single-use enrollment tokens and authenticated restart-safe long polling.

Fleet sync uses ordered revisions, state hashes, durable command delivery, and
reconciliation after reconnects. This supports safe offline operation, central
audit history, mixed-backend inventory, and verified rollback per target.

## Backend abstraction

One FirewallBackend interface; UFW and iptables adapters implement it.
Detectors resolve ownership: UFW, IPTABLES_LEGACY, IPTABLES_NFT,
FIREWALLD, NFTABLES_NATIVE, MULTIPLE_CONFLICTING, NONE, UNKNOWN.
Writes are blocked on conflict until an admin explicitly acknowledges.

## Transaction pipeline

authorize, lock, validate, conflict-check, snapshot, apply, re-read,
verify (hash compare), commit metadata, release lock. Any failure after
snapshot triggers automatic rollback; failed rollback raises a CRITICAL
notification.

## Storage

SQLite by default (pure-Go driver, WAL, tuned pragmas, single-writer
pooling). PostgreSQL optional for large fleets via DSN. No other
infrastructure required; fully offline-capable.

## Databases

Local SQLite is the tuned default (zero config). Postgres, MySQL/
MariaDB and SQLite-in-custom-path are first-class driver options.
Connection pooling, busy timeouts and integrity checks are built in.
