# Architecture

The diagram below is the implemented control-plane architecture.

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

The agent belongs on every machine whose firewall FYRwall manages. Installing
an agent beside `fyrwall-server` is supported as a deployment concept: the
central server then appears as another managed target and can manage its own
host firewall. This should be an explicit install and enrollment operation,
not silent code injection.

`fyrwall-server` routes local firewall reads and mutations through the typed
Unix socket. Remote agents enroll with single-use tokens, store a per-agent
credential, and maintain restart-safe outbound long-poll sessions. Commands and
results are persisted in SQLite so server restarts do not lose queued work.

## Target agent model

Each target agent detects the active firewall owner and routes normalized rule
transactions to the matching UFW or iptables adapter. The server sends policy
intent, never shell commands. Agent operations stay typed and allowlisted, and
each target performs validation, snapshot, apply, verification, and rollback
locally so a server disconnect cannot leave a half-applied transaction.

Agents establish outbound connections to the control plane. Managed
hosts need no inbound management port. Enrollment uses short-lived,
single-use tokens followed by per-agent client certificates plus hashed bearer
credentials. Certificates rotate automatically before expiry and can be
revoked from the Fleet UI.

## Realtime synchronization design

Near-realtime fleet state uses authenticated long polling rather than UI
polling alone. Every update carries an agent ID, monotonic sequence number,
state hash, and policy revision. Each poll refreshes the full normalized status
and rule snapshot. Sequence checks reject stale state, while persisted pending
results and timed delivery recovery reconcile reconnects. Poll cadence provides
online/offline status without mistaking silence for synchronized state.

This model enables:

- near-realtime rule, health, ownership, and agent status in one dashboard
- drift detection when local tools or automation change a target outside FYRwall
- safe offline behavior: existing firewall rules continue working while the
  control plane is unavailable
- per-target or group policy rollout with staged batches, failure budgets, and
  durable command/result tracking
- centralized audit history with target identity and local apply evidence
- version and capability negotiation across mixed distributions and backends
- fleet-wide inventory for backend type, kernel, distribution, agent version,
  health, policy revision, and last-seen time
- reusable labels and groups for sites, environments, applications, and owners
- fast incident response through scoped emergency rules and verified rollback

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

Next: [Safety and Restore Points](safety.md) · [Security Model](security.md)
