# Security Policy

FYRwall is a security tool, so its own security is treated seriously.

## Supported versions

| Version | Supported |
|---------|-----------|
| latest release | yes |
| older releases | best effort |

## Reporting a vulnerability

DO NOT open a public GitHub issue for security vulnerabilities.

Email the org contact listed at https://www.potenfyr.in/ with:

- description of the issue
- steps to reproduce or a proof of concept
- affected versions / commit
- any suggested mitigation

You will receive an acknowledgment within 72 hours. Please allow up to
90 days for a fix before public disclosure; we will credit reporters in
the release notes unless anonymity is requested.

## Scope

In scope:

- privilege escalation from the unprivileged server or agent
- shell/command injection through any API surface
- authentication or session bypass, CSRF flaws
- tamper-protection bypasses (binary hash, config encryption)
- the extension capability system (grant bypasses)

Out of scope:

- social engineering of host administrators
- vulnerabilities in the underlying Linux firewall stack itself
- reports from automated scanners without a working proof of concept

## Security design summary

- Server and agent run unprivileged; a tiny typed privileged helper is
  the only root component, with an allowlisted Unix-socket API
- No shell anywhere: argument-array execution with output caps
- Argon2id credentials, CSRF, rate limiting, RBAC enforced server-side
- Config encrypted at rest (AES-256-GCM, host keyfile 0600)
- Agent and server binaries hash-verified on boot (tamper detection)
- Logs size- and age-bounded; agent logs encrypted at rest
