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
- agent: host-networked with NET_ADMIN, talking to the real host
  firewall and exposing the typed Unix socket over a shared volume

    docker compose up -d

### How agents reach the server across the network

Agents always dial OUT to the server; managed hosts need no inbound
ports. Point each agent at the server address:

    # on each managed host
    fyrwall agent --server https://fw.example.com

Inside Docker networks, agents reach the server by service name
(server:7443). Across the internet, publish 7443 via your reverse
proxy with TLS and set server.domain so Host binding is enforced.

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
