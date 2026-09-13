# Contributing to FYRwall

Thanks for your interest in improving FYRwall. Security tooling must be
extra careful with changes, so please read this guide.

## Development setup

- Go 1.26+, bun (frontend), Docker (all tests run in containers)
- `git clone` + `cd web && bun install`
- Build: `./scripts/build.sh`
- Test: `./scripts/docker-test.sh` (never run tests on the host)

## Everyday commands

Backend:

    go build ./...                    # compile everything
    go test ./...                     # unit tests (prefer the container runner)
    ./scripts/docker-test.sh          # full suite, Docker-isolated, offline
    gofmt -l . && go vet ./...        # formatting + static checks
    gofmt -w <file>                   # apply formatting

Frontend (web UI):

    cd web && bun install             # deps
    cd web && bun run build           # production build

Docs site (this directory drives fyrwall.docs.potenfyr.in):

    cd docs && bun install            # deps
    cd docs && bun run dev            # dev server with HMR
    cd docs && bun run build          # build + prerender all routes to dist/
    cd docs && bun run typecheck      # tsc --noEmit

Docs content is plain markdown in `docs/content/*.md`; the site imports it
directly at build time, so editing a file there is all it takes. The
install script shown on the site is synced from `packaging/install.sh`
during the build.

Cross-compile and release:

    ./scripts/cross-build.sh          # all 7 arches -> dist/ (+ SHA256SUMS)
    FYRWALL_VERSION=... ./scripts/cross-build.sh

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
  (the PR template asks for this)
- CI must be green: Docker tests, 7-arch compile matrix, frontend build

## Reporting issues

Use [GitHub Issues](https://github.com/PotenFYR-Studios/FYRwall/issues) for
bugs and feature requests - there are templates for bugs, features,
docs, questions and security concerns. For anything you believe is
**exploitable**, do not open a public issue: use [private vulnerability
reporting](https://github.com/PotenFYR-Studios/FYRwall/security/advisories/new)
and see [SECURITY.md](SECURITY.md).

## License

By contributing you agree your contributions are licensed under the
Apache-2.0 with Commons Clause license covering the repository.
