# Running FYRwall in Docker

## Quick start

    docker run -d --name fyrwall \
      -p 127.0.0.1:7443:7443 \
      -v fyrwall-data:/var/lib/fyrwall \
      -v fyrwall-logs:/var/log/fyrwall \
      ghcr.io/potenfyr-studios/fyrwall:latest

Web UI: http://127.0.0.1:7443

## Compose (server + host agent bridge)

The repo ships docker-compose.yml with two services:

- server: the web/API, containerized, no special privileges
- agent: host-networked with NET_ADMIN, talking to the real host firewall and
  exposing the typed Unix socket over a shared volume

The server consumes the shared typed Unix socket. Only the agent container has
the host network namespace and firewall capabilities.

    docker compose up -d

### Remote agents

Remote agents accept a central URL and keep the local Unix socket private:

    FYRWALL_AGENT_ENROLLMENT_TOKEN='<single-use-token>' \
      fyrwall agent --server https://fw.example.com

Agents dial out, so managed hosts need no inbound management port. Production
deployments require TLS. Per-agent credentials, ordered updates, state hashes,
durable commands, and reconnect reconciliation are built in.

### Why the agent needs host network + NET_ADMIN

A container cannot see or mutate the host firewall namespaces. The
agent sidecar therefore runs with the host network namespace and
NET_ADMIN/NET_RAW caps - but still only through the typed allowlisted
socket API, never raw commands.

## Security notes

- Bind to loopback or an explicit interface, not 0.0.0.0, unless TLS
  or a proxy fronts the container
- The bundled config uses FYRWALL_ALLOW_INSECURE_BIND=true because the
  container default bind is wide; put TLS in front for real exposure
- Volumes keep the database, restore points and logs across upgrades
