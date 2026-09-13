# Operating FYRwall

## Daily workflow

1. Open the web UI at http://127.0.0.1:7443 (or your configured domain)
2. The dashboard shows health, backend, policy owner and rule count
3. Rules page: read, search and validate rules; apply changes transactionally
4. Health page: live component status
5. Notifications: persistent, deduplicated issues with remediation hints

## CLI essentials

    fyrwall status          # firewall + ownership summary
    fyrwall doctor          # read-only diagnostics
    fyrwall preflight       # diagnostics + backend detection
    fyrwall restore list    # list restore points
    fyrwall restore create  # manual restore point

## Tray and desktop integration

On desktop Linux with a tray: FYRwall appears in the system tray after
`fyrwall setup` enables it, with menu entries: Open Web UI, Restart
services, Stop, Quit. It is also registered in your application menu
and can be removed like any app (`fyrwall uninstall` or the uninstaller
script).

## Users and roles

Roles: super admin, admin, operator, auditor, viewer.
Admins manage users under Users & Access: create, disable, reset
passwords, revoke sessions. All actions are audited.

## Restore points

Created automatically before every change. Retention: 50 automatic,
20 manual by default, both configurable. Restoring shows a diff first
and snapshots current state before restoring.

Next: [Extensions](extensions.md) · [Troubleshooting](troubleshooting.md)
