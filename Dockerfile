# FYRwall production container.
# The server runs unprivileged inside the container; the agent sidecar
# (or host install) is required for firewall control on that host since
# containers cannot safely mutate the host firewall.
#
# Stage 1 builds the web GUI from source: webembed/dist contents are
# gitignored, so a Docker build must compile the frontend itself or the
# binary embeds an empty directory and the GUI loads blank.
#
# Build:  docker build -t fyrwall:latest .
# Run:    see docs/content/docker.md or README "Docker" section

# ---- Stage 1: web GUI (React/Vite) ----
FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json ./
RUN npm install --no-audit --no-fund
COPY web/ ./
RUN npm run build

# ---- Stage 2: Go binary with the GUI embedded ----
FROM golang:1.24-alpine AS build

ARG VERSION=docker
ARG COMMIT=unknown
RUN apk add --no-cache ca-certificates git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Replace (never merge) the embedded GUI with the stage-1 output so a
# stale local webembed/dist in the build context cannot leak in.
RUN rm -rf webembed/dist && mkdir -p webembed/dist
COPY --from=web /src/web/dist/ webembed/dist/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w -X github.com/PotenFYR-Studios/FYRwall/internal/version.Version=${VERSION} -X github.com/PotenFYR-Studios/FYRwall/internal/version.Commit=${COMMIT}" \
    -o /out/fyrwall ./cmd/fyrwall

FROM alpine:3.20

RUN apk add --no-cache ca-certificates iptables ip6tables \
    && addgroup -S fyrwall \
    && adduser -S -G fyrwall -h /var/lib/fyrwall fyrwall \
    && install -d -m 0750 -o fyrwall -g fyrwall /var/lib/fyrwall /var/log/fyrwall \
    && install -d -m 0750 -o root -g fyrwall /run/fyrwall /etc/fyrwall

COPY --from=build /out/fyrwall /usr/local/bin/fyrwall
COPY packaging/config.example.yaml /etc/fyrwall/config.yaml

# OCI license label (fix-round-2 W4).
LABEL org.opencontainers.image.licenses="Apache-2.0 WITH Commons-Clause-1.0"

USER fyrwall
EXPOSE 7443
VOLUME ["/var/lib/fyrwall", "/var/log/fyrwall"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:7443/api/v1/system/health >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/usr/local/bin/fyrwall"]
CMD ["server", "--config", "/etc/fyrwall/config.yaml"]

# Loopback bind inside a container blocks port mapping; default the
# container to a container-wide bind. Override in config for advanced use.
ENV FYRWALL_SERVER_BIND=0.0.0.0
# Container bind without TLS requires the explicit insecure-bind override;
# production traffic should be TLS or reverse-proxied.
ENV FYRWALL_ALLOW_INSECURE_BIND=true
