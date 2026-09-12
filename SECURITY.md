# Security Policy

FYRwall is a security tool. Reports about anything below are important to
us, and we treat them seriously.

## Supported versions

| Version | Supported |
| --- | --- |
| 0.1.x | yes |

Older releases get no patches; please update to the latest 0.1.x.

## How to report

Please use GitHub's **private vulnerability reporting** for this
repository (Security tab -> "Report a vulnerability"). This keeps details
out of public view until a fix is ready. Do not open a public issue for
anything you believe is exploitable.

For non-security bugs, use the regular issue templates instead.

## What we need in a report

- What you observed, including how to recognize the failure
- Your platform: distro, architecture, FYRwall version, firewall backend
  (UFW / iptables-legacy / iptables-nft)
- Logs relevant to the issue (redacted as below), plus config where
  relevant

Please do NOT include in reports, issues, or any public channel:

- Passwords, API tokens, session cookies, agent keys or TLS keys
- Real firewall rule dumps or network topologies you consider sensitive
  (redact hosts/ports if they matter)
- Working exploit code or proof-of-concept payloads - a clear description
  of the impact is enough at this stage

## In scope

- Authentication, session handling, RBAC and CSRF in the web UI and API
- Privilege separation of `fyrwall-server` / `fyrwall-agent` /
  `fyrwall-agent-privd`, including the Unix-socket help protocol
- Config encryption at rest (FYRCFG1), key handling, tamper detection
- The firewall transaction pipeline: validation, conflict detection,
  restore points, verify and rollback logic
- Anything that could weaken or bypass firewall state (for example a way
  to mark rules applied when they are not, or to skip rollback)
- Install path: install.sh downloads, checksum verification, systemd
  unit hardening

## Out of scope

- Vulnerabilities in supported browsers, OSes, or dependencies; report
  those upstream
- Self-inflicted lockouts from disabling safety rails in your own config
- Reports from automated scanners without a demonstrated impact

## What to expect

We will acknowledge reports as quickly as we can and keep you updated as
we investigate. Fixes land on `master` and ship in the next patch
release; you will be credited in the changelog unless you prefer
otherwise.

## Firewall safety guarantees

The product's core promise is documented in
[docs/content/safety.md](docs/content/safety.md): every change runs
through validate -> snapshot -> apply -> verify -> commit, with automatic
rollback and a critical notification if rollback itself fails. Reports
that break any of these guarantees are treated as high priority.
