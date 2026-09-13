# Releases and CI Builds

Every FYRwall release is built automatically by CI. A release ships three
ways to install — installer script, standalone tarballs, and container
images — plus SHA-256 checksums for everything.

## What a release contains

Every GitHub Release for `vX.Y.Z` carries these assets:

| Asset | Purpose |
|---|---|
| `install.sh` | Installer/updater script, byte-identical to the one served at `https://fyrwall.docs.potenfyr.in/install.sh` (both are synced from `packaging/install.sh`) |
| `uninstall.sh` | Clean removal script (`--all`, `--all --purge`) |
| `fyrwall_X.Y.Z_linux_<arch>.tar.gz` | Standalone build per architecture: binary + config example + systemd units + README + LICENSE |
| `SHA256SUMS` | SHA-256 manifest covering every asset above |

Supported architectures: amd64, arm64, arm, 386, ppc64le, s390x, riscv64.
Docker images (amd64 + arm64) are pushed to
`ghcr.io/potenfyr-studios/fyrwall` in parallel by the Docker workflow.

## How CI builds a release

| Event | What happens |
|---|---|
| version tag `vX.Y.Z` | Full release: the release is (re)created with changelog, tarballs, installers and checksums; the `:vX.Y.Z` image tag is pushed on top of the rolling tags |
| push to `master` | Same-version refresh: all release artifacts are rebuilt and **replace** the current version's release assets in place; notes are regenerated plus a single build line; container images are refreshed |
| push to `docs/**` | Docs site auto-deploys to GitHub Pages |

### Same-version replace semantics

The release pipeline is idempotent — it never creates duplicate releases:

- **Re-pushing a version tag** (`v0.1.0` again, or a moved tag): the existing
  release for that tag is deleted (assets included; the git tag itself is
  kept) and recreated from scratch. Builds and changelog are fully replaced.
- **Push to `master` while the VERSION file is unchanged**: all release
  assets are rebuilt and clobbered in place (`--clobber`), and the release
  notes are regenerated from `CHANGELOG.md` plus a single build line naming
  the commit. Repeated refreshes never accumulate build lines.
- **Push to `master` with no existing release**: nothing is published. A new
  release always requires a version-bump tag; CI logs a notice and exits
  cleanly.
- **Manual re-run**: use **Re-run all jobs** on a past run or the workflow's
  **Run workflow** button (`workflow_dispatch`).

## Installing from a release

### Installer script (recommended)

    curl -fsSL https://github.com/PotenFYR-Studios/FYRwall/releases/latest/download/install.sh | sudo sh

Review-first variant:

    curl -fLo install-fyrwall.sh https://github.com/PotenFYR-Studios/FYRwall/releases/latest/download/install.sh
    less install-fyrwall.sh
    sudo sh install-fyrwall.sh

The installer resolves the latest release, downloads the tarball for your
architecture, verifies it against `SHA256SUMS` and installs. Pinning,
offline and air-gapped installs are documented in [Installation](installation.md).

### Manual tarball install

    cd <release download directory>
    grep "fyrwall_<version>_linux_<arch>.tar.gz" SHA256SUMS | sha256sum -c -
    tar -xzf fyrwall_<version>_linux_<arch>.tar.gz
    sudo install -d -m 0750 /etc/fyrwall
    sudo install -m 0755 fyrwall_<version>_linux_<arch>/fyrwall /usr/local/bin/fyrwall
    sudo install -m 0640 fyrwall_<version>_linux_<arch>/config.example.yaml /etc/fyrwall/config.yaml
    sudo install -m 0644 fyrwall_<version>_linux_<arch>/packaging/systemd/*.service /etc/systemd/system/
    sudo systemctl daemon-reload

### Container images

    docker pull ghcr.io/potenfyr-studios/fyrwall:latest
    # pinned release tag, pushed on version-bump tags only:
    docker pull ghcr.io/potenfyr-studios/fyrwall:v0.1.0

See [Docker](docker.md) for run commands and the server + agent topology.

## Verifying a release

Every release asset is covered by `SHA256SUMS`. Verify a single downloaded
asset (the same pattern the installer uses):

    grep "fyrwall_<version>_linux_<arch>.tar.gz" SHA256SUMS | sha256sum -c -

To verify everything at once, download all assets into one directory first:

    sha256sum -c SHA256SUMS

The installer verifies the tarball checksum automatically and refuses an
unverified install. See the [security model](security.md) for how FYRwall
handles trust boundaries, and [Updating](updating.md) for the in-tool update
path.

## Version, tag and image matrix

| Version file | Tag pushed | GitHub Release | Container tags |
|---|---|---|---|
| 0.1.0, unchanged | none | assets refreshed in place, notes get a build line | `:latest`, `:0.1.0`, `:<short-sha>` refreshed |
| bumped to 0.2.0 | `v0.2.0` | new release created | `:latest`, `:0.2.0`, `:<short-sha>`, `:v0.2.0` pushed |
| bumped to 0.2.0, tag not yet pushed | none | unchanged | `:latest`, `:0.2.0`, `:<short-sha>` refreshed; **release `v0.1.0` untouched** |
| tag re-pushed / moved | `v0.1.0` | deleted and recreated (builds + changelog replaced) | same as a fresh tag build |

The raw-version image tag (`:0.1.0`) is pushed on every build, so the
rolling `:latest` and version tag always track the tip of `master`, while
`:vX.Y.Z` pins an exact release.

Next: [Installation](installation.md) · [Updating](updating.md)
