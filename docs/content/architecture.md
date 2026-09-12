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

SQLite (pure-Go driver, WAL, tuned pragmas, single-writer pooling) is the
storage engine. No external database infrastructure required; fully
offline-capable. The config schema reserves a `postgres` driver option
(with a required DSN) for an upcoming backend.

## Databases

Local SQLite is the tuned default (zero config). Connection pooling, busy
timeouts and integrity checks are built in. The config schema accepts
`postgres` as a driver value, reserved for an upcoming backend; SQLite
remains the supported engine today.
