# Extensions

Extensions add dashboard widgets, notification channels, diagnostics
probes, rule templates, event enrichers and sandboxed UI panels.

## What extensions are

A directory with extension.yaml (declarative manifest) plus optional
static assets. Installed by an administrator; every capability is
denied by default and must be granted at install time.

## What extensions can never do

No code execution on server or agent. No direct firewall mutation. No
user management, credentials, PKI or settings access. These are not
grantable - the capability does not exist in the system.

## Example manifest

    id: my-battery-widget
    name: UPS Battery Widget
    version: 1.0.0
    author: you
    capabilities:
      - dashboard.widget
    widgets:
      - id: ups-charge
        title: UPS charge
        kind: metric
        size: small
        source: http.local
        query: {"url": "http://127.0.0.1:8080/charge"}
        refresh_seconds: 30

    sudo fyrwall extension install /path/to/my-battery-widget

The install step prints the capability list and asks for explicit
grants. Denied capabilities simply do not activate.

## Distribution

Zip the directory; share it. The registry hashes all assets and shows
the digest at install so tampering is detectable.

Next: [Extensions Guide](extensions-guide.md)
