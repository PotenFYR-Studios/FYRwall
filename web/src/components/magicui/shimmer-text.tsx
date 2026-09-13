import { cn } from "../../lib/utils";

/**
 * Magic UI Shimmer (magicui.xyz/docs/components/shimmer): animated
 * gradient text shine. Used for the FYRwall wordmark and section titles.
 */
export function ShimmerText({
  children,
  className,
  duration = 2,
}: {
  children: React.ReactNode;
  className?: string;
  duration?: number;
}) {
  return (
    <span
      style={{
        // Magic UI shimmer recipe: gradient sweep over background-clip:text.
        "--speed": `${duration}s`,
        backgroundImage:
          "linear-gradient(120deg, #f97316 0%, #fdba74 40%, #f97316 60%, #38adf6 100%)",
        backgroundSize: "250% 100%",
        WebkitBackgroundClip: "text",
        backgroundClip: "text",
      } as React.CSSProperties}
      className={cn(
        "animate-[shimmer_var(--speed)_linear_infinite] text-transparent",
        className,
      )}
    >
      {children}
    </span>
  );
}
