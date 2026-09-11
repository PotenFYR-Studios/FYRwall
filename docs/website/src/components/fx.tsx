// Magic UI style ambient effects: Aurora, Particles, ShimmerButton,
// GlowingCard, AnimatedGradientText, TypingText, Marquee, Spotlight,
// BorderBeam, LiveDot. All pure CSS - zero dependencies, works fully
// offline like the rest of the site.
import React, { useEffect, useState } from "react";

/** Soft radial spotlight that follows the cursor. */
export function Spotlight({ className = "" }: { className?: string }) {
  const [pos, setPos] = useState({ x: 400, y: 200 });
  useEffect(() => {
    const onMove = (e: MouseEvent) => setPos({ x: e.clientX, y: e.clientY });
    window.addEventListener("mousemove", onMove);
    return () => window.removeEventListener("mousemove", onMove);
  }, []);
  return (
    <div
      aria-hidden
      className={`pointer-events-none absolute inset-0 overflow-hidden ${className}`}
      style={{
        background: `radial-gradient(600px circle at ${pos.x}px ${pos.y}px, rgba(249,115,22,0.10), transparent 65%)`,
      }}
    />
  );
}

/** Animated gradient beam running around the border of a card. */
export function BorderBeam({ color = "#f97316" }: { color?: string }) {
  return (
    <div aria-hidden className="pointer-events-none absolute inset-0 rounded-[inherit] overflow-hidden">
      <style>{`
        @keyframes fyr-border-beam-spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
      `}</style>
      <div
        className="absolute left-1/2 top-1/2"
        style={{
          width: "200%",
          aspectRatio: "1",
          marginLeft: "-50%",
          marginTop: "-50%",
          animation: "fyr-border-beam-spin 6s linear infinite",
          background: `conic-gradient(from 0deg, transparent 0 340deg, ${color} 350deg, transparent 360deg)`,
          mask: "radial-gradient(farthest-side, transparent calc(100% - 2px), #000 calc(100% - 2px))",
          WebkitMask:
            "radial-gradient(farthest-side, transparent calc(100% - 2px), #000 calc(100% - 2px))",
        }}
      />
    </div>
  );
}

/** Animated "live" status dot with pulse rings. */
export function LiveDot({ color = "#22c55e" }: { color?: string }) {
  return (
    <span aria-hidden className="relative inline-flex h-2.5 w-2.5">
      <span
        className="absolute inline-flex h-full w-full rounded-full opacity-60 animate-ping"
        style={{ background: color }}
      />
      <span
        className="relative inline-flex rounded-full h-2.5 w-2.5"
        style={{ background: color }}
      />
    </span>
  );
}

/** Drifting aurora blobs behind hero sections. */
export function Aurora() {
  return (
    <div aria-hidden className="pointer-events-none absolute inset-0 overflow-hidden">
      <style>{`
        @keyframes fyr-aurora-a { 0%,100% { transform: translate(-15%,-10%) scale(1); } 50% { transform: translate(10%,12%) scale(1.25); } }
        @keyframes fyr-aurora-b { 0%,100% { transform: translate(12%,-6%) scale(1.1); } 50% { transform: translate(-12%,14%) scale(0.9); } }
        @keyframes fyr-aurora-c { 0%,100% { transform: translate(0%,12%) scale(0.95); } 50% { transform: translate(-6%,-12%) scale(1.2); } }
      `}</style>
      <div className="absolute w-[42rem] h-[42rem] rounded-full opacity-25 blur-3xl"
        style={{ background: "#8b5cf6", top: "-12rem", left: "-10rem", animation: "fyr-aurora-a 14s ease-in-out infinite" }} />
      <div className="absolute w-[36rem] h-[36rem] rounded-full opacity-20 blur-3xl"
        style={{ background: "#ec4899", top: "-8rem", right: "-8rem", animation: "fyr-aurora-b 17s ease-in-out infinite" }} />
      <div className="absolute w-[30rem] h-[30rem] rounded-full opacity-20 blur-3xl"
        style={{ background: "#f97316", bottom: "-12rem", left: "30%", animation: "fyr-aurora-c 20s ease-in-out infinite" }} />
    </div>
  );
}

/** Rising particle field (tiny glowing dots). */
export function Particles({ count = 40 }: { count?: number }) {
  const dots = React.useMemo(
    () =>
      Array.from({ length: count }, () => ({
        left: Math.random() * 100,
        top: 20 + Math.random() * 80,
        size: 1 + Math.random() * 2.5,
        dur: 6 + Math.random() * 10,
        delay: -Math.random() * 12,
        hue: Math.random() > 0.5 ? "#f97316" : "#8b5cf6",
      })),
    [count]
  );
  return (
    <div aria-hidden className="pointer-events-none absolute inset-0 overflow-hidden">
      <style>{`
        @keyframes fyr-particle-float {
          0% { transform: translateY(0) scale(1); opacity: 0; }
          15% { opacity: 0.9; }
          85% { opacity: 0.5; }
          100% { transform: translateY(-120px) scale(0.4); opacity: 0; }
        }
      `}</style>
      {dots.map((d, i) => (
        <span
          key={i}
          className="absolute rounded-full"
          style={{
            left: `${d.left}%`,
            top: `${d.top}%`,
            width: d.size,
            height: d.size,
            background: d.hue,
            boxShadow: `0 0 ${d.size * 3}px ${d.hue}`,
            animation: `fyr-particle-float ${d.dur}s ease-in-out ${d.delay}s infinite`,
          }}
        />
      ))}
    </div>
  );
}

/** Button with animated conic shimmer border + hover glow. */
export function ShimmerButton({
  href,
  children,
  variant = "primary",
}: {
  href: string;
  children: React.ReactNode;
  variant?: "primary" | "ghost";
}) {
  if (variant === "ghost") {
    return (
      <a
        href={href}
        className="relative rounded-lg border border-zinc-700 px-6 py-3 font-semibold text-zinc-200
                   hover:bg-zinc-900 hover:border-zinc-500 transition-all duration-300 hover:-translate-y-0.5
                   hover:shadow-[0_8px_30px_rgba(0,0,0,0.4)]"
      >
        {children}
      </a>
    );
  }
  return (
    <a href={href} className="relative group rounded-lg p-[1.5px] overflow-hidden">
      <style>{`
        @keyframes fyr-shimmer-spin { to { transform: rotate(360deg); } }
      `}</style>
      <span
        aria-hidden
        className="absolute inset-[-150%]"
        style={{
          background: "conic-gradient(from 0deg, transparent 0deg, #f97316 60deg, #ec4899 120deg, transparent 180deg)",
          animation: "fyr-shimmer-spin 3.5s linear infinite",
        }}
      />
      <span
        className="relative block rounded-[6.5px] bg-orange-600 group-hover:bg-orange-500 px-6 py-3 font-semibold text-white
                   transition-all duration-300 group-hover:-translate-y-0.5 group-hover:shadow-[0_10px_40px_-10px_rgba(249,115,22,0.7)]"
      >
        {children}
      </span>
    </a>
  );
}

/** Card with soft glowing border that intensifies on hover. */
export function GlowingCard({
  children,
  className = "",
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={`relative group rounded-xl ${className}`}>
      <div
        aria-hidden
        className="absolute inset-0 rounded-xl opacity-0 group-hover:opacity-100 transition-opacity duration-500 blur-xl"
        style={{ background: "linear-gradient(135deg, rgba(249,115,22,0.25), rgba(139,92,246,0.2))" }}
      />
      <div className="relative rounded-xl border border-zinc-800 bg-zinc-900/80 group-hover:border-orange-500/40 transition-colors duration-500">
        {children}
      </div>
    </div>
  );
}

/** Text with a slow animated gradient sweep. */
export function AnimatedGradientText({ children }: { children: React.ReactNode }) {
  return (
    <span className="relative inline-block">
      <style>{`
        @keyframes fyr-gradient-x {
          0%, 100% { background-position: 0% 50%; }
          50% { background-position: 100% 50%; }
        }
      `}</style>
      <span
        className="bg-clip-text text-transparent"
        style={{
          backgroundImage: "linear-gradient(90deg, #f97316, #ec4899, #8b5cf6, #f97316)",
          backgroundSize: "300% 100%",
          animation: "fyr-gradient-x 8s ease infinite",
        }}
      >
        {children}
      </span>
    </span>
  );
}

/** Typewriter effect that cycles through phrases. */
export function TypingText({ phrases, className = "" }: { phrases: string[]; className?: string }) {
  const [idx, setIdx] = useState(0);
  const [text, setText] = useState("");
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    const full = phrases[idx % phrases.length];
    if (!deleting && text === full) {
      const t = setTimeout(() => setDeleting(true), 1600);
      return () => clearTimeout(t);
    }
    if (deleting && text === "") {
      setDeleting(false);
      setIdx((i) => (i + 1) % phrases.length);
      return;
    }
    const t = setTimeout(
      () => setText(deleting ? full.slice(0, text.length - 1) : full.slice(0, text.length + 1)),
      deleting ? 28 : 58
    );
    return () => clearTimeout(t);
  }, [text, deleting, idx, phrases]);

  return (
    <span className={className}>
      {text}
      <span className="text-orange-500 animate-pulse">▍</span>
    </span>
  );
}

/** Infinite horizontal marquee. Duplicate children for seamless loop. */
export function Marquee({ items }: { items: string[] }) {
  return (
    <div className="relative overflow-hidden py-2" style={{ maskImage: "linear-gradient(90deg, transparent, #000 12%, #000 88%, transparent)", WebkitMaskImage: "linear-gradient(90deg, transparent, #000 12%, #000 88%, transparent)" }}>
      <style>{`
        @keyframes fyr-marquee { from { transform: translateX(0); } to { transform: translateX(-50%); } }
      `}</style>
      <div className="flex gap-3 w-max" style={{ animation: "fyr-marquee 28s linear infinite" }}>
        {[...items, ...items].map((it, i) => (
          <span
            key={i}
            className="whitespace-nowrap text-xs text-zinc-400 border border-zinc-800 bg-zinc-900/80 rounded-full px-3 py-1"
          >
            {it}
          </span>
        ))}
      </div>
    </div>
  );
}
