# Installation

## One-liner (recommended)

    curl -fsSL https://fyrwall.docs.potenfyr.in/install.sh | sudo sh

The script resolves the latest release, downloads the tarball for your
architecture, verifies SHA256SUMS and installs. Air-gapped hosts: download
the tarball yourself, extract it next to the script, and run the script
locally (it prefers local binaries).

Review-first (safer):

    curl -fsSLo install-fyrwall.sh https://fyrwall.docs.potenfyr.in/install.sh
    less install-fyrwall.sh
    sudo sh install-fyrwall.sh

## What the installer does

1. Checks for Linux and maps the architecture (amd64, arm64, arm, 386, ppc64le, s390x, riscv64)
2. Creates the unprivileged `fyrwall` service account
3. Installs the binary: local tarball first, then `dist/fyrwall`, then a checksum-verified release download
4. Installs the config (existing config is never overwritten) and registers the desktop entry
5. Installs the hardened systemd units
6. Verifies checksums when available
7. Runs non-destructive preflight checks

The installer NEVER touches your firewall rules.

## Pin a version or stay offline

    FYRWALL_VERSION=0.1.0 sudo -E sh install.sh
    FYRWALL_NO_DOWNLOAD=1 sudo -E sh install.sh   # local tarball or dist/fyrwall only

## Manual install from a release tarball

    tar -xzf fyrwall_0.1.0_linux_amd64.tar.gz
    sudo install -d -m 0750 /etc/fyrwall
    sudo install -m 0755 fyrwall_0.1.0_linux_amd64/fyrwall /usr/local/bin/fyrwall
    sudo install -m 0640 -o root -g root fyrwall_0.1.0_linux_amd64/config.example.yaml /etc/fyrwall/config.yaml
    sudo install -m 0644 fyrwall_0.1.0_linux_amd64/packaging/systemd/*.service /etc/systemd/system/
    sudo systemctl daemon-reload

## Verify checksums

    cd <release download directory>
    sha256sum -c SHA256SUMS

## First boot

    sudo systemctl enable --now fyrwall-agent fyrwall-server
    # open http://127.0.0.1:7443: the one-time setup wizard creates the
    # super admin account and its password right in the browser.

See [Operation](operation.md) for the daily workflow.
