// Local Magic UI components, adapted from potenfyr-nest patterns to be
// dependency-free (pure CSS / rAF) like the rest of the site. Landing
// page only, per SPEC section 7 (no effects inside doc articles).
import { useEffect, useId, useRef, useState } from "react";

/** Magic UI - Number Ticker: counts up to `value` (rAF). */
export function NumberTicker({
  value,
  className,
}: {
  value: number;
  className?: string;
}) {
  const [display, setDisplay] = useState(0);
  const prev = useRef(0);

  useEffect(() => {
    if (
      typeof document !== "undefined" &&
      (document.hidden ||
        window.matchMedia("(prefers-reduced-motion: reduce)").matches)
    ) {
      prev.current = value;
      setDisplay(value);
      return;
    }
    const from = prev.current;
    const start = performance.now();
    const duration = 900;
    let raf = 0;
    const step = (t: number) => {
      const p = Math.min(1, (t - start) / duration);
      const eased = 1 - Math.pow(1 - p, 3);
      setDisplay(Math.round(from + (value - from) * eased));
      if (p < 1) raf = requestAnimationFrame(step);
      else prev.current = value;
    };
    raf = requestAnimationFrame(step);
    return () => cancelAnimationFrame(raf);
  }, [value]);

  return (
    <span className={className} aria-label={String(value)}>
      {display}
    </span>
  );
}

/** Magic UI - Marquee: seamless infinite scroller, pauses on hover. */
export function Marquee({
  items,
  reverse = false,
  duration = 30,
}: {
  items: string[];
  reverse?: boolean;
  duration?: number;
}) {
  return (
    <div
      className="marquee"
      style={{
        maskImage:
          "linear-gradient(90deg, transparent, #000 12%, #000 88%, transparent)",
        WebkitMaskImage:
          "linear-gradient(90deg, transparent, #000 12%, #000 88%, transparent)",
      }}
    >
      {[0, 1].map((half) => (
        <div
          key={half}
          aria-hidden={half === 1}
          className="marquee-track"
          style={{ animationDirection: reverse ? "reverse" : "normal", animationDuration: `${duration}s` }}
        >
          {items.map((it, i) => (
            <span key={i} className="chip mono-label">
              {it}
            </span>
          ))}
        </div>
      ))}
    </div>
  );
}

/** Magic UI - Magic Card: cursor spotlight that tracks the mouse. */
export function MagicCard({
  children,
  className = "",
  gradientColor = "rgba(249,115,22,0.10)",
}: {
  children: React.ReactNode;
  className?: string;
  gradientColor?: string;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ x: -400, y: -400 });
  const [inside, setInside] = useState(false);

  return (
    <div
      ref={ref}
      onMouseMove={(e) => {
        const r = ref.current?.getBoundingClientRect();
        if (r) setPos({ x: e.clientX - r.left, y: e.clientY - r.top });
      }}
      onMouseEnter={() => setInside(true)}
      onMouseLeave={() => setInside(false)}
      className={`relative overflow-hidden rounded-2xl border border-line-light bg-white/[0.02] transition-colors duration-300 hover:border-brand-violet/50 ${className}`}
    >
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 transition-opacity duration-300"
        style={{
          background: `radial-gradient(340px circle at ${pos.x}px ${pos.y}px, ${gradientColor}, transparent 70%)`,
          opacity: inside ? 1 : 0,
        }}
      />
      {children}
    </div>
  );
}

/** Magic UI - Dot Pattern: decorative dotted backdrop (SVG). */
export function DotPattern({ className = "" }: { className?: string }) {
  const id = useId();
  return (
    <svg
      aria-hidden
      className={`pointer-events-none absolute inset-0 h-full w-full ${className}`}
    >
      <defs>
        <pattern id={id} width="28" height="28" patternUnits="userSpaceOnUse">
          <circle cx="2" cy="2" r="1.2" fill="rgba(139,92,246,0.18)" />
        </pattern>
      </defs>
      <rect width="100%" height="100%" fill={`url(#${id})`} />
    </svg>
  );
}

/** Magic UI - Meteors: deterministic streaking comets (hero only). */
export function Meteors({ number = 14 }: { number?: number }) {
  const meteors = Array.from({ length: number }, (_, i) => ({
    id: i,
    left: (i * 137) % 100,
    delay: ((i * 2.3) % 8).toFixed(1),
    dur: (4.5 + ((i * 1.1) % 4)).toFixed(1),
  }));
  return (
    <div aria-hidden className="pointer-events-none absolute inset-0 overflow-hidden">
      {meteors.map((m) => (
        <span
          key={m.id}
          className="meteor"
          style={{
            left: `${m.left}%`,
            animationDuration: `${m.dur}s`,
            animationDelay: `${m.delay}s`,
          }}
        />
      ))}
    </div>
  );
}

/** Ambient glow orb for hero atmosphere (SPEC 5.14). */
export function GlowOrb({
  className = "",
  color = "rgba(139, 92, 246, 0.18)",
  size = 420,
}: {
  className?: string;
  color?: string;
  size?: number;
}) {
  return (
    <div
      aria-hidden
      className={`orb ${className}`}
      style={{ width: size, height: size, background: color }}
    />
  );
}
