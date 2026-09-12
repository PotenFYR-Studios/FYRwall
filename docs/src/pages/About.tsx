// About: mission, guarantees, org and licensing - all from the README.
import { Link } from "react-router-dom";
import { ShieldCheck, Github, MessageSquare, Globe } from "lucide-react";

const GUARANTEES = [
  "The web/API server runs unprivileged. Only the tiny local agent touches the firewall, through a strictly permissioned Unix socket.",
  "Every privileged operation is a strongly typed, allowlisted request. There is no generic exec endpoint and no shell string anywhere in the codebase.",
  "Every firewall mutation follows the full pipeline: authorize, lock, validate, detect conflicts, snapshot, apply, re-read, verify, and automatic rollback on any failure after the snapshot step.",
  "Firewall ownership detection blocks writes whenever multiple independent managers are active. Nothing is ever disabled silently.",
  "Failed or suspicious changes roll back automatically. Rollback failure raises a critical notification, never a silent success.",
];

export default function About() {
  return (
    <div className="dot-backdrop">
      <div className="mx-auto max-w-3xl px-6 pb-20 pt-14">
        <p className="eyebrow">
          <span className="eyebrow-accent">About</span> the project
        </p>
        <h1 className="grad-text mt-3 text-[clamp(2.2em,5vw,3.2em)] font-extrabold leading-[1.08] tracking-tight">
          Firewalls deserve better tooling
        </h1>
        <p className="mt-4 text-[1.04em] leading-[1.75] text-muted">
          FYRwall is a safe, focused, modern web-based firewall
          administration platform for Linux. It gives administrators a clean
          GUI over UFW and iptables without ever running the web server as
          root, without shell interpolation anywhere, and without a single
          firewall mutation that lacks a restore point and rollback path.
        </p>

        <h2 className="mt-10 border-b border-line pb-2 text-[1.32em] font-bold text-white">
          Core guarantees
        </h2>
        <ul className="mt-4 space-y-3">
          {GUARANTEES.map((g) => (
            <li key={g.slice(0, 24)} className="flex gap-3 text-[0.95em] text-ink-2">
              <ShieldCheck className="mt-1 h-4 w-4 shrink-0 text-brand-emerald" />
              <span>{g}</span>
            </li>
          ))}
        </ul>

        <h2 className="mt-10 border-b border-line pb-2 text-[1.32em] font-bold text-white">
          PotenFYR Studios
        </h2>
        <p className="mt-4 text-[0.95em] text-ink-2">
          FYRwall is built by PotenFYR Studios, a small open-source studio.
          The org keeps its tools open and auditable under Apache-2.0 with
          the Commons Clause: free to use, self-host and embed, but nobody
          resells it as-is.
        </p>
        <div className="mt-5 grid gap-[14px] sm:grid-cols-3">
          <a
            href="https://github.com/PotenFYR-Studios"
            className="doc-card"
            target="_blank"
            rel="noopener noreferrer"
          >
            <span className="card-icon">
              <Github className="h-5 w-5 text-brand-violet" />
            </span>
            <span className="card-title">GitHub org</span>
            <span className="card-desc">Source, issues and releases.</span>
          </a>
          <a
            href="https://potenfyr.in/"
            className="doc-card"
            target="_blank"
            rel="noopener noreferrer"
          >
            <span className="card-icon">
              <Globe className="h-5 w-5 text-brand-cyan" />
            </span>
            <span className="card-title">potenfyr.in</span>
            <span className="card-desc">The studio's home on the web.</span>
          </a>
          <a
            href="https://discord.com/invite/zUaN2FPBec"
            className="doc-card"
            target="_blank"
            rel="noopener noreferrer"
          >
            <span className="card-icon">
              <MessageSquare className="h-5 w-5 text-brand-pink" />
            </span>
            <span className="card-title">Support Discord</span>
            <span className="card-desc">Questions, help and updates.</span>
          </a>
        </div>

        <h2 className="mt-10 border-b border-line pb-2 text-[1.32em] font-bold text-white">
          License
        </h2>
        <p className="mt-4 text-[0.95em] text-ink-2">
          FYRwall is licensed under the <strong className="text-ink">Apache License 2.0
          with the Commons Clause</strong>: free to use, study, modify,
          self-host and redistribute, and embedding it inside a larger
          product is welcome. Selling FYRwall itself as a paid product or
          managed host is the one bright line. See{" "}
          <Link to="/docs/license">the license page</Link> - and the{" "}
          <a
            href="https://github.com/PotenFYR-Studios/FYRwall/blob/master/LICENSE"
            target="_blank"
            rel="noopener noreferrer"
          >
            LICENSE
          </a>{" "}
          file, which is authoritative.
        </p>

        <p className="mt-12 text-center font-mono text-[0.78em] text-faint">
          Made with &lt;3 by PotenFYR Studios
        </p>
      </div>
    </div>
  );
}
