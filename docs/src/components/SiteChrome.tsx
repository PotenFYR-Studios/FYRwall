// Site chrome: 56px sticky blur header (SPEC 5.1) and the full-bleed
// 3-zone footer (SPEC 5.2). Mobile drawer below 900px.
import { useState } from "react";
import { Link, NavLink } from "react-router-dom";
import { Menu, X, Search } from "lucide-react";

const NAV = [
  { to: "/", label: "Home", end: true },
  { to: "/docs", label: "Docs", end: false },
  { to: "/examples", label: "Examples", end: false },
  { to: "/about", label: "About", end: false },
];

const EXTERNAL = [
  { href: "https://potenfyr.in/", label: "Website" },
  { href: "https://discord.com/invite/zUaN2FPBec", label: "Discord" },
  { href: "https://github.com/PotenFYR-Studios/FYRwall", label: "GitHub" },
];

function BrandMark() {
  return (
    <img
      src="/favicon.png"
      alt=""
      width={24}
      height={24}
      className="brand-mark"
    />
  );
}

export function SiteHeader() {
  const [open, setOpen] = useState(false);

  return (
    <header className="site-header">
      <Link to="/" className="brand" aria-label="FYRwall documentation home">
        <BrandMark />
        <span>
          FYRwall<span className="brand-dot">.</span>
          <span className="mono-label text-muted">docs</span>
        </span>
      </Link>

      <nav className="ml-4 hidden items-center gap-1 md:flex" aria-label="Primary">
        {NAV.map((n) => (
          <NavLink
            key={n.to}
            to={n.to}
            end={n.end}
            className={({ isActive }) => `nav-link${isActive ? " active" : ""}`}
          >
            {n.label}
          </NavLink>
        ))}
      </nav>

      <div className="ml-auto hidden items-center gap-2 md:flex">
        <button
          type="button"
          className="flex items-center gap-2 rounded-lg border border-line-light bg-white/[0.03] px-3 py-1.5 text-xs text-muted transition-colors hover:border-brand-violet/50"
          onClick={() => window.dispatchEvent(new CustomEvent("fyrwall:open-palette"))}
          aria-label="Search documentation"
        >
          <Search className="h-3.5 w-3.5" />
          <span>Search</span>
          <kbd>Ctrl K</kbd>
        </button>
        {EXTERNAL.map((e) => (
          <a
            key={e.label}
            href={e.href}
            className="header-ext"
            target="_blank"
            rel="noopener noreferrer"
          >
            {e.label}
          </a>
        ))}
      </div>

      <button
        type="button"
        className="ml-auto inline-flex items-center rounded-lg border border-line-light bg-white/[0.03] p-2 text-muted md:hidden"
        aria-expanded={open}
        aria-label="Toggle navigation menu"
        onClick={() => setOpen((v) => !v)}
      >
        {open ? <X className="h-4 w-4" /> : <Menu className="h-4 w-4" />}
      </button>

      {open && (
        <div
          className="absolute inset-x-0 top-full z-40 border-b border-line-light bg-[#0b0d14]/98 px-4 py-3 backdrop-blur md:hidden"
          style={{ backgroundColor: "rgba(11,13,20,0.98)" }}
        >
          <nav className="flex flex-col gap-1" aria-label="Mobile">
            {NAV.map((n) => (
              <NavLink
                key={n.to}
                to={n.to}
                end={n.end}
                onClick={() => setOpen(false)}
                className={({ isActive }) => `nav-link${isActive ? " active" : ""}`}
              >
                {n.label}
              </NavLink>
            ))}
            <div className="my-2 h-px bg-white/5" />
            {EXTERNAL.map((e) => (
              <a
                key={e.label}
                href={e.href}
                className="nav-link"
                target="_blank"
                rel="noopener noreferrer"
              >
                {e.label}
              </a>
            ))}
          </nav>
        </div>
      )}
    </header>
  );
}

export function SiteFooter() {
  return (
    <footer className="site-footer">
      <div className="sf-inner">
        <div className="sf-brand">
          <Link to="/" className="brand">
            <BrandMark />
            <span>
              FYRwall<span className="brand-dot">.</span>docs
            </span>
          </Link>
          <p className="sf-tagline">
            Safe, focused, web-based firewall administration for Linux.
            Open source, offline-capable, unprivileged by design.
          </p>
        </div>
        <div className="sf-links">
          <a href="https://github.com/PotenFYR-Studios" target="_blank" rel="noopener noreferrer">
            GitHub Org
          </a>
          <a href="https://potenfyr.in/" target="_blank" rel="noopener noreferrer">
            potenfyr.in
          </a>
          <a href="https://discord.com/invite/zUaN2FPBec" target="_blank" rel="noopener noreferrer">
            Support Discord
          </a>
          <a href="https://github.com/PotenFYR-Studios/FYRwall" target="_blank" rel="noopener noreferrer">
            Project home
          </a>
          <Link to="/docs" className="accent">
            Docs
          </Link>
          <Link to="/docs/license">
            License
          </Link>
        </div>
      </div>
      <div className="sf-legal">
        <span>
          © 2026 PotenFYR Studios. Released under Apache-2.0 with Commons
          Clause. The <a href="https://github.com/PotenFYR-Studios/FYRwall/blob/master/LICENSE">LICENSE</a> file is authoritative.
        </span>
        <span>Crafted with &lt;3 for Linux administrators.</span>
      </div>
    </footer>
  );
}
