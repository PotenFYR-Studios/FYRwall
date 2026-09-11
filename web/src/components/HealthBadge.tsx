import { useQuery } from "@tanstack/react-query";

async function fetchJSON<T>(url: string): Promise<T> {
  const res = await fetch(url, { credentials: "same-origin" });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    const msg = body?.error?.message ?? res.statusText;
    throw new Error(msg);
  }
  return res.json();
}

export { fetchJSON };

export function HealthBadge() {
  const { data, isLoading } = useQuery({
    queryKey: ["health"],
    queryFn: () => fetchJSON<{ state: string }>("/api/v1/system/health"),
    refetchInterval: 15_000,
  });
  if (isLoading || !data) return null;
  const color =
    data.state === "HEALTHY"
      ? "bg-green-500/20 text-green-400"
      : data.state === "DEGRADED"
        ? "bg-yellow-500/20 text-yellow-400"
        : "bg-red-500/20 text-red-400";
  return (
    <span className={`text-xs px-2 py-1 rounded-full ${color}`} data-testid="health-state">
      {data.state}
    </span>
  );
}
