# Installation

## One-liner (recommended)

    curl -fsSL https://potenfyr-studios.github.io/FYRwall/install.sh | sudo sh

Review-first (safer):

    curl -fsSLo install-fyrwall.sh https://potenfyr-studios.github.io/FYRwall/install.sh
    less install-fyrwall.sh
    sudo sh install-fyrwall.sh

## What the installer does

1. Detects Linux, architecture, distro and init system
2. Creates the unprivileged `fyrwall` service account
3. Creates /etc/fyrwall, /var/lib/fyrwall, /var/log/fyrwall with strict permissions
4. Installs the binary, config and systemd units
5. Runs non-destructive preflight checks
6. NEVER touches your firewall rules

## Interactive setup

Run `sudo fyrwall setup` after install. The wizard asks:

- Bind address and port (default 127.0.0.1:7443)
- Enable TLS now or terminate TLS at a reverse proxy
- Domain binding (optional, for internet-facing GUIs)
- Auto-start on boot (yes/no)
- Automatic restore points before every change (recommended: yes)
- Update notifications (off by default; opt in)
- Setup mode: guided scan (we detect everything) or manual config
- First admin username; password is read from the environment, never argv

## Manual install from release tarball

    tar -xzf fyrwall_0.1.0_linux_amd64.tar.gz
    cd fyrwall_0.1.0_linux_amd64
    sudo ./install.sh

## Verify checksums

    sha256sum -c SHA256SUMS
