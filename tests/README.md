# Local test harness

## run-server.sh — host the FYRwall server from this checkout

Builds the web GUI, syncs it into `webembed/dist`, builds the Go binary,
and serves everything on `http://127.0.0.1:7443` with an isolated config
(sqlite, logs and integrity state under `tests/.tmp/`, gitignored).
Nothing outside the repo is touched; Ctrl-C stops the server.

```sh
tests/run-server.sh                  # build if stale, serve on :7443
tests/run-server.sh --port 8080      # different port
tests/run-server.sh --no-build       # reuse the binary from the last run
tests/run-server.sh --rebuild-frontend
```

On first run (empty database) the web GUI opens the one-time super admin
setup wizard: choose the admin username and password there (policy: 14+
chars with upper, lower, digit and special). Delete `tests/.tmp/` for a
factory reset.

> The web GUI lives in the Go binary via `webembed/dist`. That directory's
> contents are gitignored, so **any** Go-only build (without the frontend
> sync this script performs) ships a blank GUI. The release pipeline and
> the Dockerfile both build the frontend for this reason.

## Dockerfile.test — unit and integration tests

```sh
docker build -f Dockerfile.test -t fyrwall-test:latest .
docker run --rm --network=none --read-only fyrwall-test:latest
```
