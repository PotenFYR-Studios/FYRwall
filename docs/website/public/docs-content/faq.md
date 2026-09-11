# FAQ

**Is FYRwall really free?**
Yes. Apache-2.0 with Commons Clause: use, modify, self-host freely and
embed it inside larger products. What nobody may do is sell FYRwall
itself as a standalone paid product.

**How is this different from Cockpit/Webmin?**
FYRwall is firewall-focused with transactional safety: conflict
detection, lockout prevention, restore points and verified rollback.
General-purpose panels manage many things; FYRwall does one thing
safely.

**Does it work behind nginx/Caddy?**
Yes. Set trusted_proxies and your domain. The GUI enforces the
configured Host when domain binding is on.

**Can it manage many servers?**
Yes: agents connect out to a central server (mTLS enrollment planned
in the fleet roadmap; single-host and LAN modes work today).

**What if the agent dies?**
The web server stays up, shows the agent offline, disables writes and
gives recovery steps. Firewall rules on the host are untouched.

**Does it send data anywhere?**
No. Telemetry, analytics and update checks are off by default. When
you enable update checks, the only request is one manifest GET.

**Which distros?**
Debian/Ubuntu, RHEL/Rocky/Alma, Fedora, SUSE, Arch, Alpine, Void,
Gentoo - anything Go binaries run on. glibc and musl. Seven CPU
architectures.

**Can I break my firewall with it?**
The product is designed so that is hard: risky changes are blocked,
every apply has a snapshot + verify + rollback, and safe-apply timers
undo connectivity-breaking changes automatically.

**Uninstall cleanly?**
    sudo sh uninstall.sh --all --purge
Removes binaries, units, user, data, logs. Your firewall rules are
never touched by FYRwall install or uninstall.

**Can I use FYRwall commercially?**
Yes, with one rule. You may embed FYRwall as a feature inside a larger
product or service, including paid ones. You may not sell FYRwall
itself: no charging for the software, no paid managed hosting of
FYRwall alone, no paid support whose value is FYRwall itself.

**Where do I put the license notice?**
Keep the LICENSE file and attribution in your distribution. If you
redistribute, your license notice must include the Commons Clause
condition alongside Apache-2.0.
