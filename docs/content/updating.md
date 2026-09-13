# Updating

## In-tool notification

Enable update checks in Settings or config:

    updates:
      check_enabled: true
      manifest_url: https://fyrwall.docs.potenfyr.in/updates.json
      check_interval_hours: 24

When a new version is released, a persistent notification appears in the
web UI with the changelog link. Disabled by default (no telemetry).

## Manual update (recommended path)

    sudo fyrwall update          # or:
    curl -fsSL https://fyrwall.docs.potenfyr.in/install.sh | sudo sh

The installer/updater: creates a database backup and firewall restore
point, downloads the matching artifact, verifies SHA256SUMS, replaces the
binary, runs migrations, restarts services, runs post-update health checks.
Your config is never overwritten.

### Release builds stay current

CI keeps GitHub Releases fresh: while the VERSION file is unchanged, every
push to `master` rebuilds the release assets in place and regenerates the
notes (a single build line per version, never accumulating); a version-bump
tag creates the next release. The one-liner therefore always fetches the
latest verified build. Full replace semantics:
[Releases and CI Builds](releases.md).

## Rollback a bad update

Restore points and DB backups are kept. Reinstall the previous release
tarball from GitHub Releases (checksums on file) and restart.

## Update checks are opt-in

FYRwall update checks are off by default - no telemetry. When enabled, the
only outbound request is the manifest GET.

Next: [Releases and CI Builds](releases.md) · [Troubleshooting](troubleshooting.md)
