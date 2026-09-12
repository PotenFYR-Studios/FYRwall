// Landing page (SPEC: hero + orbs + meteors + marquee + capability and
// stat tiles). Every claim mirrors the repository README - nothing
// invented. Effects are confined to the hero per SPEC section 7.
import { Link } from "react-router-dom";
import {
  ShieldCheck,
  LockKeyhole,
  Globe,
  Zap,
  Radar,
  Puzzle,
  ArrowUpRight,
  Terminal,
} from "lucide-react";
import { NumberTicker, Marquee, Meteors, GlowOrb, BorderBeam, MagicCard } from "../components/magicui";
import { CopyButton } from "../components/buttons";

const INSTALL_CMD = "curl -fsSL https://fyrwall.docs.potenfyr.in/install.sh | sudo sh";

const FEATURES: Array<{
  icon: React.ReactNode;
  title: string;
  body: string;
  tint: string;
}> = [
  {
    icon: <ShieldCheck className="h-5 w-5" />,
    title: "Never locked out",
    body: "Risky changes are blocked before they apply; every apply has a restore point, SHA-256 verification and automatic rollback.",
    tint: "text-brand-emerald",
  },
  {
    icon: <LockKeyhole className="h-5 w-5" />,
    title: "Unprivileged by design",
    body: "The web server never runs as root. A tiny typed agent socket with an allowlisted operation set is the only path to netfilter.",
    tint: "text-brand-violet",
  },
  {
    icon: <Globe className="h-5 w-5" />,
    title: "Any Linux, any arch",
    body: "Debian to Alpine, amd64 to riscv64. Fully offline-capable, no telemetry, ever.",
    tint: "text-brand-cyan",
  },
  {
    icon: <Zap className="h-5 w-5" />,
    title: "Transactional applies",
    body: "Authorize, lock, validate, detect conflicts, snapshot, apply, re-read, verify, rollback. Every single time.",
    tint: "text-brand-orange",
  },
  {
    icon: <Radar className="h-5 w-5" />,
    title: "Ownership detection",
    body: "UFW, iptables-legacy/nft, firewalld, native nftables: conflicting managers block writes safely. Nothing is disabled silently.",
    tint: "text-brand-pink",
  },
  {
    icon: <Puzzle className="h-5 w-5" />,
    title: "Extensible",
    body: "Dashboard widgets, notification channels, probes and rule templates via sandboxed, capability-scoped manifests.",
    tint: "text-brand-violet",
  },
];

const PIPELINE = [
  "authorize",
  "lock",
  "validate",
  "conflict check",
  "snapshot",
  "apply",
  "re-read",
  "verify",
  "rollback on failure",
];

const MARQUEE = [
  "UFW", "iptables-legacy", "iptables-nft", "nftables", "firewalld detection",
  "systemd", "OpenRC", "runit", "s6", "amd64", "arm64", "arm", "386",
  "ppc64le", "s390x", "riscv64", "Debian", "Ubuntu", "Fedora", "Arch", "Alpine",
];

const OWNERSHIP_STATES = [
  { label: "UFW", warn: false },
  { label: "IPTABLES_LEGACY", warn: false },
  { label: "IPTABLES_NFT", warn: false },
  { label: "FIREWALLD", warn: false },
  { label: "NFTABLES_NATIVE", warn: false },
  { label: "MULTIPLE_CONFLICTING", warn: true },
  { label: "NONE", warn: false },
  { label: "UNKNOWN", warn: false },
];

const STATS: Array<[number, string, string, string]> = [
  [7, "release architectures", "text-brand-violet", "Cpu"],
  [46, "tests in CI, Docker-isolated", "text-brand-pink", "FlaskConical"],
  [0, "root processes in the UI path", "text-brand-emerald", "ShieldCheck"],
  [8, "distros in the compat matrix", "text-brand-cyan", "Container"],
];

function StatIcon({ name, className }: { name: string; className?: string }) {
  const icons: Record<string, React.ReactNode> = {
    Cpu: <Cpu className={className} />,
    FlaskConical: <FlaskConical className={className} />,
    ShieldCheck: <ShieldCheck className={className} />,
    Container: <Container className={className} />,
  };
  return <>{icons[name]}</>;
}

import { Cpu, FlaskConical, Container } from "lucide-react";

export default function Home() {
  return (
    <div className="dot-backdrop">
      {/* hero */}
      <div className="relative overflow-hidden">
        <GlowOrb className="-top-40 left-[8%]" color="rgba(139,92,246,0.18)" size={520} />
        <GlowOrb className="-top-32 right-[6%]" color="rgba(236,72,153,0.15)" size={460} />
        <GlowOrb className="top-[280px] left-1/2 -translate-x-1/2" color="rgba(6,182,212,0.10)" size={380} />
        <Meteors number={16} />
        <section className="relative mx-auto max-w-5xl px-6 pb-14 pt-[72px] text-center">
          <div className="hero-in" style={{ animationDelay: "0s" }}>
            <span className="status-pill mx-auto">
              <span className="pulse-dot" />
              v0.1.0 / open source / Apache-2.0 with Commons Clause
            </span>
          </div>
          <h1
            className="hero-in grad-text mx-auto mt-6 max-w-4xl text-[clamp(2.6em,6vw,4em)] font-extrabold leading-[1.08] tracking-tight"
            style={{ animationDelay: "0.08s" }}
          >
            The safe way to manage Linux firewalls
          </h1>
          <p
            className="hero-in mx-auto mt-5 max-w-[720px] text-[1.04em] leading-[1.75] text-muted"
            style={{ animationDelay: "0.16s" }}
          >
            Transactional firewall changes with automatic rollback, lockout
            protection and zero root web processes. UFW, iptables-legacy and
            iptables-nft behind one GUI - self-hosted and offline-capable.
          </p>
          <div
            className="hero-in mt-8 flex flex-wrap justify-center gap-4"
            style={{ animationDelay: "0.24s" }}
          >
            <Link to="/docs/installation" className="btn btn-primary">
              Get started
            </Link>
            <a
              href="https://github.com/PotenFYR-Studios/FYRwall"
              className="btn btn-ghost"
              target="_blank"
              rel="noopener noreferrer"
            >
              View source
            </a>
          </div>
          <div className="hero-in relative mx-auto mt-10 max-w-2xl" style={{ animationDelay: "0.32s" }}>
            <div className="relative rounded-xl border border-line-light bg-panel/80 p-1 shadow-[var(--shadow)] backdrop-blur">
              <BorderBeam duration={8} colorFrom="#8b5cf6" colorTo="#f97316" />
              <div className="flex items-center gap-3 rounded-[10px] px-4 py-3">
                <Terminal className="h-4 w-4 shrink-0 text-brand-orange" />
                <code className="min-w-0 flex-1 overflow-x-auto whitespace-nowrap font-mono text-sm text-[#dfe2ef]">
                  {INSTALL_CMD}
                </code>
                <CopyButton text={INSTALL_CMD} />
              </div>
            </div>
            <p className="mt-3 font-mono text-[11px] uppercase tracking-[0.14em] text-faint">
              review-first: download the script, read it, then run it
            </p>
          </div>
        </section>
      </div>

      <div className="mx-auto max-w-7xl px-6 pb-24">
        {/* marquee */}
        <Marquee items={MARQUEE} />

        {/* features */}
        <section className="mt-16 grid gap-[14px] md:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map((f) => (
            <div key={f.title} className="doc-card">
              <span className="card-icon">
                <span className={f.tint}>{f.icon}</span>
              </span>
              <span className="card-title mt-1">{f.title}</span>
              <span className="card-desc">{f.body}</span>
            </div>
          ))}
        </section>

        {/* capability cards: backends + ownership states */}
        <section className="mt-20">
          <p className="eyebrow">
            <span className="eyebrow-accent">Capability</span> cards
          </p>
          <h2 className="mt-2 text-[1.6em] font-bold tracking-tight text-white">
            One normalized rule model, every backend
          </h2>
          <p className="mt-2 max-w-2xl text-[0.95em] text-muted">
            Adapters translate UFW and iptables (legacy and nft flavor) into a
            single validated rule model. Ownership detectors resolve who owns
            the firewall before a single write is allowed.
          </p>
          <div className="mt-6 grid gap-[14px] sm:grid-cols-2 lg:grid-cols-4">
            <MagicCard className="p-5">
              <span className="tag">adapter</span>
              <h3 className="mt-3 font-semibold text-white">UFW</h3>
              <p className="card-desc mt-1">
                Parses status verbose and numbered output, v6 sections, LIMIT
                rules, multiport specs and parenthesized comments.
              </p>
            </MagicCard>
            <MagicCard className="p-5">
              <span className="tag">adapter</span>
              <h3 className="mt-3 font-semibold text-white">iptables legacy</h3>
              <p className="card-desc mt-1">
                iptables-save parser with multiport, conntrack and quoted
                comments; iptables-restore powers rollback.
              </p>
            </MagicCard>
            <MagicCard className="p-5">
              <span className="tag">adapter</span>
              <h3 className="mt-3 font-semibold text-white">iptables-nft</h3>
              <p className="card-desc mt-1">
                Same parser family with automatic flavor detection between
                legacy and nft backends.
              </p>
            </MagicCard>
            <MagicCard className="p-5">
              <span className="tag">detection</span>
              <h3 className="mt-3 font-semibold text-white">Ownership engine</h3>
              <p className="card-desc mt-1">
                firewalld and native nftables rulesets are detected; writes
                block on conflicts until an admin acknowledges.
              </p>
            </MagicCard>
          </div>
          <div className="mt-6 flex flex-wrap items-center gap-2">
            <span className="mono-label text-faint">ownership states:</span>
            {OWNERSHIP_STATES.map((s) =>
              s.warn ? (
                <span key={s.label} className="status-pill warn">{s.label}</span>
              ) : (
                <span key={s.label} className="tag">{s.label}</span>
              )
            )}
          </div>
        </section>

        {/* pipeline */}
        <section className="relative mt-20 overflow-hidden rounded-2xl border border-line-light bg-white/[0.02] p-6 backdrop-blur">
          <BorderBeam duration={9} colorFrom="#ec4899" colorTo="#8b5cf6" />
          <h2 className="text-center text-[1.15em] font-semibold text-white">
            Every firewall change walks the full pipeline
          </h2>
          <div className="mt-5 flex flex-wrap justify-center gap-2">
            {PIPELINE.map((step, i) => (
              <span
                key={step}
                className={`chip${i === PIPELINE.length - 1 ? " !border-amber-500/30 !text-amber-300" : ""}`}
              >
                <span className="mr-2 font-mono text-[0.9em] text-faint">{i + 1}</span>
                {step}
              </span>
            ))}
          </div>
          <p className="mt-4 text-center text-[0.83em] text-muted">
            Any failure after the snapshot triggers automatic rollback; a
            failed rollback raises a critical notification, never a silent
            success.
          </p>
        </section>

        {/* stats */}
        <section className="mx-auto mt-20 grid max-w-3xl grid-cols-2 gap-3.5 sm:grid-cols-4">
          {STATS.map(([value, label, tint, icon]) => (
            <div
              key={label}
              className="rounded-2xl border border-white/10 bg-white/[0.02] p-4 text-center backdrop-blur-md transition-transform hover:-translate-y-0.5"
            >
              <div className={`mx-auto mb-2 flex justify-center ${tint}`}>
                <StatIcon name={icon} className="h-5 w-5" />
              </div>
              <div className="grad-text ticker font-mono text-3xl font-extrabold">
                <NumberTicker value={value} />
              </div>
              <div className="mt-1 text-[11.5px] uppercase tracking-[1.4px] text-muted">
                {label}
              </div>
            </div>
          ))}
        </section>

        {/* docs teaser */}
        <section className="mt-20">
          <p className="eyebrow">
            <span className="eyebrow-accent">Read</span> the docs
          </p>
          <div className="mt-4 grid gap-[14px] md:grid-cols-3">
            <Link to="/docs/installation" className="doc-card">
              <span className="card-title">Installation</span>
              <span className="card-desc">
                One-liner, review-first and tarball installs on any distro.
              </span>
              <ArrowUpRight className="card-arrow h-4 w-4" />
            </Link>
            <Link to="/docs/safety" className="doc-card">
              <span className="card-title">Safety and restore points</span>
              <span className="card-desc">
                The transaction pipeline, verification and rollback guarantees.
              </span>
              <ArrowUpRight className="card-arrow h-4 w-4" />
            </Link>
            <Link to="/docs/security" className="doc-card">
              <span className="card-title">Security model</span>
              <span className="card-desc">
                Privilege separation, Argon2id, RBAC and encrypted logs.
              </span>
              <ArrowUpRight className="card-arrow h-4 w-4" />
            </Link>
          </div>
        </section>

        {/* cta */}
        <section className="mt-20 text-center">
          <h2 className="grad-text text-[clamp(1.8em,4vw,2.6em)] font-extrabold tracking-tight">
            Safe by default
          </h2>
          <p className="mx-auto mt-3 max-w-xl text-[0.95em] text-muted">
            Single Go binary, embedded UI, no Node.js on production hosts,
            config encrypted at rest. If it can lock you out, FYRwall blocks
            it first.
          </p>
          <div className="mt-7 flex flex-wrap justify-center gap-4">
            <Link to="/docs/installation" className="btn btn-primary">
              Install FYRwall
            </Link>
            <Link to="/examples" className="btn btn-ghost">
              Browse examples
            </Link>
          </div>
        </section>
      </div>
    </div>
  );
}
