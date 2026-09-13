import { motion } from "framer-motion";

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
  return (
    <motion.div
      initial={{ opacity: 0, filter: "blur(6px)", y: 8 }}
      animate={{ opacity: 1, filter: "blur(0px)", y: 0 }}
      transition={{ delay, duration }}
      className={cn(className)}
    >
      {children}
    </motion.div>
  );
}
