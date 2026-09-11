# Contributing to FYRwall

Thanks for your interest in improving FYRwall. Security tooling must be
extra careful with changes, so please read this guide.

## Development setup

- Go 1.24+, bun (frontend), Docker (all tests run in containers)
- `git clone` + `cd web && bun install`
- Build: `./scripts/build.sh`
- Test: `./scripts/docker-test.sh` (never run tests on the host)

## Ground rules

1. **No shell interpolation anywhere.** All subprocess calls go through
   `internal/system/exec` with argument arrays.
2. **No new privileged operations without review.** The agent API is a
   strictly allowlisted, typed surface.
3. **Firewall mutations must be transactional.** Snapshot, verify,
   rollback - any new write path must preserve this.
4. **No secrets in code, logs, or tests.** Passwords come from the
   environment; tokens are hashed at rest.
5. **Capability-based detection** over distro-name checks.
6. ASCII only in source and docs; no em/en dashes or section signs.
7. Run `gofmt` and `go vet` before every commit.

## Pull requests

- One logical change per PR; include tests for behavior changes
- Parsers need fixtures first (including malformed input cases)
- Explain the security implications of any change touching `internal/agent`,
  `internal/auth`, `internal/firewall/manager.go`, or `internal/secureconfig`
- CI must be green: Docker tests, 7-arch compile matrix, frontend build

## Reporting vulnerabilities

See [SECURITY.md](SECURITY.md) - never open public issues for
vulnerabilities.

## License

By contributing you agree your contributions are licensed under the
Apache-2.0 WITH Commons Clause license covering the repository.
