# Safety and Restore Points

FYRwall treats every firewall mutation as a transaction. The same pipeline
runs for every change, every time:

    authorize, lock, validate, conflict check, snapshot, apply, re-read,
    verify (hash compare), commit metadata, release lock

Any failure after the snapshot step triggers automatic rollback. A failed
rollback raises a critical notification - it is never reported as a silent
success.

## Structural validation

Before anything touches the firewall, the rule model is checked for CIDR
errors, invalid ports, family mismatches, interface names and loopback
safety. Conflicts are reported with severity levels and remediation hints:

- duplicate rules
- shadowed rules
- allow/deny overlap
- SSH lockout risk
- management-port lockout

Critical conflicts (SSH lockout, loopback denial) hard-block the apply at
the validation layer.

## Ownership gating

FYRwall detects who owns the firewall: UFW, iptables-legacy, iptables-nft,
firewalld, native nftables, multiple conflicting managers, none or unknown.
Writes are blocked on conflict until an admin explicitly acknowledges.
Nothing is ever disabled silently.

## Restore points

Every change gets an automatic restore point with SHA-256 state hashes.
Apply verification re-reads the firewall and compares hashes; a mismatch
rolls back automatically.

Retention defaults: 50 automatic and 20 manual restore points, 365 days.
Tune with `restore_points.retain_automatic`, `retain_manual` and
`retention_days`.

Create a manual point with `fyrwall restore create`, list with
`fyrwall restore list`. Restoring from the web UI shows a diff first and
snapshots current state before restoring.

## Safe apply timer

Connectivity-breaking changes are undone automatically after
`firewall.safe_apply_timeout_seconds` (60 by default), so an admin can reach
the host over SSH to confirm or adjust.

## When things go wrong

Next: [Security Model](security.md) · [Troubleshooting](troubleshooting.md)
