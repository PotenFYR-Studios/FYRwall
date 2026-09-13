import { useReducedMotion, motion } from "framer-motion";

import { cn } from "../../lib/utils";

/**
 * Magic UI BlurFade (magicui.xyz/docs/components/blur-fade): content
 * fades+unblurs into view. Page/section entrance.
 */
export function BlurFade({
  children,
  className,
  delay = 0,
  duration = 0.4,
}: {
  children: React.ReactNode;
  className?: string;
  delay?: number;
  duration?: number;
}) {
  const reducedMotion = useReducedMotion();

  return (
    <motion.div
      initial={reducedMotion ? { opacity: 0 } : { opacity: 0, filter: "blur(6px)", y: 8 }}
      animate={reducedMotion ? { opacity: 1 } : { opacity: 1, filter: "blur(0px)", y: 0 }}
      transition={reducedMotion ? { delay, duration: 0.2 } : { delay, duration }}
      className={cn(className)}
    >
      {children}
    </motion.div>
  );
}
