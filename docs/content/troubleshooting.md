# Troubleshooting

## Web UI unreachable

    systemctl status fyrwall-server
    fyrwall doctor

Check the bind address (default 127.0.0.1 only) and port conflicts.

## Writes blocked (FW_BACKEND_CONFLICT)

Multiple firewall managers are active (e.g. ufw + firewalld). Disable all
but one, press Refresh. Resolution is detected and notifications
auto-resolve.

## I locked myself out over SSH

A change that risks SSH lockout is blocked by default. If you used Safe
Apply: the rollback timer restored the previous state after the configured
timeout. Otherwise use console access:

    fyrwall restore list

List restore points, then restore from the web UI (it shows a diff first and
snapshots current state before restoring). Never reboot blindly - you can
lose the restore path.

## Agent offline

    systemctl status fyrwall-agent
    fyrwall doctor

Agent reconnects with exponential backoff and spools events locally.

## Disk usage

All logs are size- and age-bounded. Check Settings for retention; default
ceiling is ~300 MiB per component.

## Password reset

Password changes and resets are managed from the web UI under Users &
Access by an administrator. There is no standalone reset-password CLI
command; if you cannot reach the UI, use console access and the web UI on
the host.
