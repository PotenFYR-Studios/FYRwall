import { motion, AnimatePresence } from "framer-motion";

import { cn } from "../../lib/utils";

/**
 * Magic UI AnimatedList (magicui.xyz/docs/components/animated-list):
 * notifications stack in from the bottom with staggered spring motion.
 */
export function AnimatedList({
  children,
  className,
  delay = 300,
  ...props
}: {
  children: React.ReactNode[];
  className?: string;
  delay?: number;
} & React.HTMLAttributes<HTMLDivElement>) {
  const items = Array.isArray(children) ? children : [children];
  return (
    <div className={cn("flex flex-col gap-3", className)} {...props}>
      <AnimatePresence initial={false}>
        {items.map((child, i) => (
          <motion.div
            key={(child as { key?: React.Key })?.key ?? i}
            initial={{ scale: 0.9, opacity: 0, y: 24 }}
            animate={{ scale: 1, opacity: 1, y: 0 }}
            exit={{ scale: 0.95, opacity: 0 }}
            transition={{
              type: "spring",
              stiffness: 350,
              damping: 40,
              delay: (items.length - 1 - i) * (delay / 1000),
            }}
          >
            {child}
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  );
}
