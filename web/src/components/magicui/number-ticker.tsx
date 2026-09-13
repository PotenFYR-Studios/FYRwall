import { useEffect, useRef } from "react";
import { useInView, useMotionValue, useReducedMotion, useSpring } from "framer-motion";

import { cn } from "../../lib/utils";

/**
 * Magic UI NumberTicker (magicui.xyz/docs/components/number-ticker):
 * counts up to the value when it enters the viewport. Dashboard stats.
 */
export function NumberTicker({
  value,
  className,
  delay = 0,
}: {
  value: number;
  className?: string;
  delay?: number;
}) {
  const ref = useRef<HTMLSpanElement>(null);
  const reducedMotion = useReducedMotion();
  const motionValue = useMotionValue(0);
  const springValue = useSpring(motionValue, { damping: 60, stiffness: 100 });
  const isInView = useInView(ref, { once: true, margin: "0px" });

  useEffect(() => {
    if (!isInView) return;

    if (reducedMotion) {
      motionValue.jump(value);
      return;
    }

    const timeout = setTimeout(() => motionValue.set(value), delay * 1000);
    return () => clearTimeout(timeout);
  }, [motionValue, isInView, reducedMotion, delay, value]);

  useEffect(() => {
    if (reducedMotion && ref.current) {
      ref.current.textContent = Intl.NumberFormat("en-US").format(value);
      return;
    }

    return springValue.on("change", (latest) => {
      if (ref.current) {
        ref.current.textContent = Intl.NumberFormat("en-US").format(Math.round(latest));
      }
    });
  }, [reducedMotion, springValue, value]);

  return (
    <span ref={ref} className={cn("inline-block tabular-nums", className)}>
      0
    </span>
  );
}
