# Updating

## In-tool notification

Enable update checks in Settings or config:

    updates:
      check_enabled: true
      manifest_url: https://potenfyr-studios.github.io/FYRwall/updates.json
      check_interval_hours: 24

When a new version is released, a persistent notification appears in
the web UI with the changelog link. Disabled by default (no telemetry).

## Manual update (recommended path)

    sudo fyrwall update          # or:
    curl -fsSL https://potenfyr-studios.github.io/FYRwall/install.sh | sudo sh

The installer/updater: creates a database backup and firewall restore
point, downloads the matching artifact, verifies SHA256SUMS, replaces
the binary, runs migrations, restarts services, runs post-update
health checks. Your config is never overwritten.

## Rollback a bad update

Restore points and DB backups are kept; reinstall the previous
tarball from GitHub Releases - checksums on file - and restart.

## Auto-update (optional, off by default)

    sudo fyrwall setup set autoupdate=true
    sudo fyrwall setup set autoupdate_window="sun 03:00"

Auto-updates only apply verified releases within the maintenance
window and always create a restore point first.
