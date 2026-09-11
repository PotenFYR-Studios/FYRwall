import {
  Spotlight,
  BorderBeam,
  LiveDot,
  Aurora,
  Particles,
  ShimmerButton,
  GlowingCard,
  AnimatedGradientText,
  TypingText,
  Marquee,
} from "../components/fx";

const features: [string, string, string][] = [
  ["🔒", "Never locked out", "Every risky change is blocked or auto-rolled back with restore points and SHA-256 verified state."],
  ["🛡️", "Unprivileged by design", "The web server never runs as root. Typed agent socket, zero shell, zero interpolation."],
  ["🌍", "Any Linux, any arch", "Debian to Alpine, amd64 to riscv64. Fully offline-capable, no telemetry, ever."],
  ["⚡", "Transactional applies", "Authorize, lock, validate, snapshot, apply, verify, rollback. Every single time."],
  ["🧭", "Ownership detection", "UFW, iptables-legacy/nft, firewalld, nftables - conflicting managers block writes safely."],
  ["🧩", "Extensible", "Dashboard widgets, notification channels, probes and templates via sandboxed manifests."],
];

const pipeline = [
  "authorize", "lock", "validate", "detect conflicts", "snapshot",
  "apply", "re-read", "verify", "auto-rollback on failure",
];

const marqueeItems = [
  "UFW", "iptables-legacy", "iptables-nft", "nftables", "firewalld detection",
  "systemd", "OpenRC", "runit", "s6", "amd64", "arm64", "arm", "386",
  "ppc64le", "s390x", "riscv64", "Debian", "Ubuntu", "Fedora", "Arch", "Alpine",
];

export default function Home() {
  return (
    <div className="relative space-y-14">
      <Spotlight />

      {/* Hero */}
      <section className="relative text-center py-16 overflow-hidden rounded-2xl border border-zinc-800 bg-zinc-950/60">
        <Aurora />
        <Particles count={36} />
        <div className="relative px-4">
          <div className="inline-flex items-center gap-2 text-xs text-zinc-400 border border-zinc-800 rounded-full px-3 py-1 bg-zinc-900/80 backdrop-blur">
            <LiveDot /> v0.1.0 - open source, Apache-2.0 with Commons Clause
          </div>
          <h1 className="mt-5 text-4xl md:text-5xl font-bold tracking-tight">
            The <AnimatedGradientText>safe</AnimatedGradientText> way to manage Linux firewalls
          </h1>
          <p className="mt-4 text-zinc-400 text-lg max-w-2xl mx-auto min-h-[3.5rem]">
            <TypingText
              phrases={[
                "Transactional firewall changes with auto-rollback.",
                "Lockout protection built in.",
                "UFW | iptables | nftables - one GUI.",
                "Unprivileged by design. No shell. No root UI.",
                "7 architectures. Every distro. Offline-first.",
              ]}
            />
          </p>
          <div className="mt-7 flex gap-4 justify-center flex-wrap">
            <ShimmerButton href="#/docs/installation">Get Started</ShimmerButton>
            <ShimmerButton href="https://github.com/PotenFYR-Studios/FYRwall" variant="ghost">
              View Source
            </ShimmerButton>
          </div>
          <div className="mt-8">
            <div className="inline-block relative rounded-lg">
              <BorderBeam color="#f97316" />
              <code className="relative block bg-zinc-900 border border-zinc-800 rounded-lg px-4 py-3 text-sm text-orange-300">
                curl -fsSL https://fyrwall.docs.potenfyr.in/install.sh | sudo sh
              </code>
            </div>
          </div>
        </div>
      </section>

      {/* Marquee of supported tech */}
      <Marquee items={marqueeItems} />

      {/* Feature grid */}
      <section className="grid md:grid-cols-3 gap-4">
        {features.map(([icon, title, body]) => (
          <GlowingCard key={title} className="p-5">
            <div className="text-2xl">{icon}</div>
            <h2 className="mt-2 font-semibold text-orange-400">{title}</h2>
            <p className="mt-2 text-sm text-zinc-400">{body}</p>
          </GlowingCard>
        ))}
      </section>

      {/* Safety pipeline strip */}
      <section className="relative rounded-2xl border border-zinc-800 bg-zinc-900/50 p-6 overflow-hidden">
        <BorderBeam color="#8b5cf6" />
        <h2 className="text-center text-lg font-semibold text-zinc-200">
          Every firewall change walks the full pipeline
        </h2>
        <div className="mt-5 flex flex-wrap justify-center gap-2 text-xs md:text-sm">
          {pipeline.map((step, i) => (
            <span
              key={step}
              className={`rounded-full px-3 py-1 border transition-all duration-300 hover:-translate-y-0.5 ${
                i === 8
                  ? "border-red-800 text-red-300 bg-red-950/40"
                  : "border-zinc-700 text-zinc-300 bg-zinc-900 hover:border-orange-500/50"
              }`}
            >
              {step}
            </span>
          ))}
        </div>
      </section>

      {/* Stats strip */}
      <section className="grid grid-cols-2 md:grid-cols-4 gap-4 text-center">
        {[
          ["7", "release architectures"],
          ["46+", "tests in CI, Docker-isolated"],
          ["0", "root processes in the UI path"],
          ["100%", "offline capable"],
        ].map(([n, label]) => (
          <GlowingCard key={label} className="p-5">
            <div className="text-3xl font-bold text-orange-500">{n}</div>
            <div className="mt-1 text-xs text-zinc-400">{label}</div>
          </GlowingCard>
        ))}
      </section>
    </div>
  );
}
