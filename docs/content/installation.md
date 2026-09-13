# Installation

FYRwall is Linux-only (systemd recommended, works without it). Three
supported install methods, all served from GitHub Releases; every method
installs the same single static binary and never touches your firewall
rules.

## Requirements

- Linux (any distro with glibc or musl; the binary is static,
  CGO_ENABLED=0)
- amd64, arm64, arm, 386, ppc64le, s390x or riscv64
- curl or wget (installer only)
- Optional: systemd for the hardened units, a desktop environment for the
  tray/app-menu entry

## Method 1 — installer script (recommended)

### One-liner

    curl -fsSL https://fyrwall.docs.potenfyr.in/install.sh | sudo sh

Same script, straight from GitHub Releases:

    curl -fsSL https://github.com/PotenFYR-Studios/FYRwall/releases/latest/download/install.sh | sudo sh

### Review-first (safer)

    curl -fLo install-fyrwall.sh https://github.com/PotenFYR-Studios/FYRwall/releases/latest/download/install.sh
    less install-fyrwall.sh
    sudo sh install-fyrwall.sh

(The docs-domain URL `https://fyrwall.docs.potenfyr.in/install.sh` serves
the identical file.)

### What the installer does

1. Checks for Linux and maps the architecture (amd64, arm64, arm, 386,
   ppc64le, s390x, riscv64)
2. Creates the unprivileged `fyrwall` service account
3. Installs the binary: local tarball first, then `dist/fyrwall`, then a
   checksum-verified release download
4. Installs the config (existing config is never overwritten) and registers
   the desktop entry
5. Installs the hardened systemd units
6. Verifies checksums when available
7. Runs non-destructive preflight checks

The installer NEVER touches your firewall rules. Uninstall:
`sudo sh packaging/uninstall.sh --all [--purge]` or `fyrwall uninstall`.

### Installer environment variables

| Variable | Default | Effect |
|---|---|---|
| `FYRWALL_VERSION` | latest release | Pin a version: `FYRWALL_VERSION=0.1.0` (with or without the `v` prefix) |
| `FYRWALL_PREFIX` | `/usr/local` | Install prefix; binaries go to `$FYRWALL_PREFIX/bin` (`/usr/bin` when prefix is `/usr`) |
| `FYRWALL_NO_DOWNLOAD` | unset | Set to `1` to forbid release downloads; local tarball or `dist/fyrwall` only (air-gap enforcement) |

Pinned or offline install:

    FYRWALL_VERSION=0.1.0 sudo -E sh install.sh
    FYRWALL_NO_DOWNLOAD=1 sudo -E sh install.sh   # local tarball or dist/fyrwall only

### Air-gapped hosts

On a connected machine, download `install.sh`, the tarball for your arch
and `SHA256SUMS` from the
[latest release](https://github.com/PotenFYR-Studios/FYRwall/releases/latest),
copy all three into one directory on the target host, then:

    tar -xzf fyrwall_<version>_linux_<arch>.tar.gz
    grep "fyrwall_<version>_linux_<arch>.tar.gz" SHA256SUMS | sha256sum -c -
    FYRWALL_NO_DOWNLOAD=1 FYRWALL_VERSION=<version> sudo -E sh install.sh

The installer prefers the extracted `fyrwall_<version>_linux_<arch>/`
directory next to the script. `sudo -E` is required so the variables
survive sudo's env_reset; pin `FYRWALL_VERSION` because an offline
directory has no VERSION file to read.

## Method 2 — manual tarball from GitHub Releases

Grab assets from the
[latest release](https://github.com/PotenFYR-Studios/FYRwall/releases/latest):

    curl -fLO https://github.com/PotenFYR-Studios/FYRwall/releases/latest/download/fyrwall_0.1.0_linux_amd64.tar.gz
    curl -fLO https://github.com/PotenFYR-Studios/FYRwall/releases/latest/download/SHA256SUMS
    grep "fyrwall_0.1.0_linux_amd64.tar.gz" SHA256SUMS | sha256sum -c -

    tar -xzf fyrwall_0.1.0_linux_amd64.tar.gz
    sudo install -d -m 0750 /etc/fyrwall
    sudo install -m 0755 fyrwall_0.1.0_linux_amd64/fyrwall /usr/local/bin/fyrwall
    sudo install -m 0640 fyrwall_0.1.0_linux_amd64/config.example.yaml /etc/fyrwall/config.yaml
    sudo install -m 0644 fyrwall_0.1.0_linux_amd64/packaging/systemd/*.service /etc/systemd/system/
    sudo systemctl daemon-reload

(Adjust the version/arch in the filenames; `uname -m` tells you yours —
x86_64 → amd64, aarch64 → arm64.)

## Method 3 — Docker

    docker run -d --name fyrwall \
      -p 127.0.0.1:7443:7443 \
      -v fyrwall-data:/var/lib/fyrwall \
      ghcr.io/potenfyr-studios/fyrwall:latest

Full guide including the server + host-agent topology:
[Docker](docker.md).

## First boot

    sudo systemctl enable --now fyrwall-agent fyrwall-server
    # open http://127.0.0.1:7443: the one-time setup wizard creates the
    # super admin account and its password right in the browser.

Without systemd, run `sudo fyrwall agent` and `fyrwall server` (or the tray:
`fyrwall tray`). Re-running any install method is a safe in-place upgrade.

Next: [Configuration](configuration.md) · [Operation](operation.md) · [Updating](updating.md)
