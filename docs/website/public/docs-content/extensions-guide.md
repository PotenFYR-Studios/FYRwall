# Writing FYRwall Extensions - Developer Guide

## Structure

    my-extension/
      extension.yaml     # manifest (required)
      assets/            # optional static files for panels

## Manifest reference

    id: acme-dash            # lowercase-dash, unique
    name: ACME Dashboard
    version: 1.2.0
    author: Acme
    homepage: https://acme.example
    min_app: 0.1.0
    capabilities:            # every entry needs admin grant at install
      - dashboard.widget
      - notify.channel
      - diagnostics.check
      - rules.template
      - events.enrich
      - ui.panel
      - firewall.status.read
      - events.read

    widgets:                 # dashboard cards, schema-driven
      - id: wan-load
        title: WAN load
        kind: metric         # metric | gauge | table | status
        size: medium         # small | medium | wide
        source: firewall.status
        refresh_seconds: 30

    notify_channels:         # typed outbound only
      - id: ops-webhook
        type: webhook        # webhook | ntfy | gotify | discord
        url: https://hooks.example/fyrwall
        min_severity: warning

    checks:                  # declarative probes, no commands
      - id: portal-http
        name: Portal reachable
        type: http           # http | tcp | file
        target: https://portal.example/health
        timeout_seconds: 5
        expect: 200

    templates:               # extra rule template packs
      - id: acme-baseline
        name: ACME baseline
        description: Standard office rules
        rules_json: '[{"direction":"in","action":"allow","protocol":"tcp","destination_port":"443"}]'

    panels:                  # sandboxed static UI (iframe, no same-origin)
      - id: acme-panel
        title: ACME panel
        entry: assets/panel.html
        sandbox: allow-scripts

## Install

    sudo fyrwall extension install ./my-extension
    # prompts y/N per capability
    sudo fyrwall extension list

## Hard rules the installer enforces

- Unknown or forbidden capabilities are rejected outright (exec,
  firewall.write, users.manage, credentials.read, pki.sign and friends
  can never be granted)
- Panels may not request iframe allow-same-origin
- Channel types limited to webhook, ntfy, gotify, discord
- Check types limited to http, tcp, file - no command execution
- Asset digest is computed and displayed at install for tamper evidence
