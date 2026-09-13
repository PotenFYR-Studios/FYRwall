import { useId } from "react";
import { motion } from "framer-motion";

import { cn } from "../../lib/utils";

/**
 * Magic UI DotPattern (magicui.design/docs/components/dot-pattern).
 * Subtle background texture for hero/auth screens.
 */
export function DotPattern({
  width = 20,
  height = 20,
  cx = 1,
  cy = 1,
  cr = 1,
  className,
  ...props
}: {
  width?: number;
  height?: number;
  cx?: number;
  cy?: number;
  cr?: number;
  className?: string;
} & React.SVGProps<SVGSVGElement>) {
  const id = useId();
  return (
    <svg aria-hidden="true" className={cn("pointer-events-none absolute inset-0 h-full w-full fill-neutral-400/40", className)} {...props}>
      <defs>
        <pattern id={id} width={width} height={height} patternUnits="userSpaceOnUse" patternContentUnits="userSpaceOnUse">
          <circle cx={cx} cy={cy} r={cr} />
        </pattern>
      </defs>
      <rect width="100%" height="100%" strokeWidth={0} fill={`url(#${id})`} />
    </svg>
  );
}

/**
 * Magic UI BorderBeam: a light beam travelling around a card's border.
 * Used on the primary dashboard/hero cards.
 */
export function BorderBeam({
  className,
  size = 50,
  duration = 6,
  colorFrom = "#f97316",
  colorTo = "#38adf6",
  delay = 0,
}: {
  className?: string;
  size?: number;
  duration?: number;
  colorFrom?: string;
  colorTo?: string;
  delay?: number;
}) {
  return (
    <div className="pointer-events-none absolute inset-0 rounded-[inherit] border border-transparent [mask-clip:padding-box,border-box] [mask-composite:intersect] [mask-image:linear-gradient(transparent,transparent),linear-gradient(#000,#000)]">
      <motion.div
        className={cn("absolute aspect-square bg-gradient-to-l from-transparent via-0 to-transparent", className)}
        style={{
          width: size,
          offsetPath: `rect(0 auto auto 0 round ${size}px)`,
          "--color-from": colorFrom,
          "--color-to": colorTo,
        } as React.CSSProperties}
        initial={{ offsetDistance: "0%" }}
        animate={{ offsetDistance: "100%" }}
        transition={{
          repeat: Infinity,
          ease: "linear",
          duration,
          delay: -delay,
        }}
      >
        <div className="h-full w-full" style={{ background: `linear-gradient(to left, ${colorFrom}, ${colorTo}, transparent)` }} />
      </motion.div>
    </div>
  );
}
