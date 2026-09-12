<!-- Thank you for contributing to FYRwall!
     Keep PRs focused; one logical change per PR is easiest to review.
     Never include secrets (passwords, tokens, keys) in code, logs or
     screenshots. -->

## Summary

<!-- What does this PR change and why? Link issues with "Fixes #123". -->

## Type of change

- [ ] Bug fix
- [ ] New feature
- [ ] Documentation
- [ ] Refactor (no behavior change)
- [ ] Build / CI / packaging

## Security impact

Does this PR touch any of the following? Check all that apply; explain
briefly under "Security notes".

- [ ] `internal/agent/` or the priv-helper Unix socket protocol
- [ ] `internal/auth/` or session / RBAC / CSRF handling
- [ ] `internal/firewall/manager.go` or the apply / rollback pipeline
- [ ] `internal/secureconfig/` or key handling
- [ ] install.sh, systemd units or packaging
- [ ] None of the above

**Security notes:**

<!-- For checked boxes: what changed, what could break, how you verified
     the safety property still holds (validation, snapshot, verify,
     rollback, least privilege, no new secrets in logs). -->

## Testing

<!-- How was this tested? e.g. `scripts/docker-test.sh`, `go test ./...`,
     manual steps on <distro>, docs build (`cd docs && bun run build`). -->

- [ ] `go test ./...` (or `scripts/docker-test.sh`) passes
- [ ] Frontend builds (`cd web && bun run build`) - if UI changed
- [ ] Docs build (`cd docs && bun run build`) - if docs changed

## Checklist

- [ ] Commits follow the repo's style; no secrets committed
- [ ] Docs updated where behavior changed
- [ ] ASCII-only text in source and docs (CONTRIBUTING rule 6)
- [ ] No new telemetry or outbound network calls added
