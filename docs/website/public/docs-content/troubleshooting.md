# Troubleshooting

## Web UI unreachable

    systemctl status fyrwall-server
    fyrwall doctor
Check bind address (default 127.0.0.1 only) and port conflicts.

## Writes blocked (FW_BACKEND_CONFLICT)

Multiple firewall managers are active (e.g. ufw + firewalld). Disable
all but one, press Refresh. Resolution is detected and notifications
auto-resolve.

## I locked myself out over SSH

A change that risks SSH lockout is blocked by default. If you used
Safe Apply: the rollback timer restored the previous state after the
configured timeout. Otherwise use console access:

    fyrwall restore list
    fyrwall restore apply <id>

## Agent offline

    systemctl status fyrwall-agent
    fyrwall doctor
Agent reconnects with exponential backoff and spools events locally.

## Disk usage

All logs are size- and age-bounded. Check Settings for retention;
default ceiling is ~300 MiB per component.

## Password reset

    FYRWALL_ADMIN_PASSWORD='new-long-password' sudo -E fyrwall user reset-password admin
