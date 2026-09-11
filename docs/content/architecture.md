# Architecture

    Browser (React, embedded)
        |
        v
    fyrwall-server (unprivileged)
        |  Unix socket /run/fyrwall/agent.sock (0660)
        v
    fyrwall-agent (unprivileged) --> privileged helper (root, typed ops)
        |
        v
    UFW / iptables / nftables

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
