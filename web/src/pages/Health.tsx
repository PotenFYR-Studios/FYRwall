import { useQuery } from "@tanstack/react-query";
import { fetchJSON } from "../components/HealthBadge";

type Component = { name: string; status: string; detail?: string; checked: string };

export default function Health() {
  const { data } = useQuery({
    queryKey: ["health-full"],
    queryFn: () =>
      fetchJSON<{ state: string; components: Component[]; uptime_s: number }>(
        "/api/v1/system/health"
      ),
    refetchInterval: 10_000,
  });
  if (!data) return <p className="text-zinc-500">Loading...</p>;
  return (
    <div className="space-y-3">
      <h2 className="text-lg font-semibold">
        System health: <span data-testid="overall-state">{data.state}</span>
      </h2>
      <ul className="space-y-2">
        {data.components.map((c) => (
          <li key={c.name} className="rounded border border-zinc-800 bg-zinc-900 p-3 flex items-center gap-3">
            <span
              className={`text-xs px-2 py-0.5 rounded ${
                c.status === "HEALTHY"
                  ? "bg-green-500/20 text-green-400"
                  : c.status === "DEGRADED"
                    ? "bg-yellow-500/20 text-yellow-400"
                    : "bg-red-500/20 text-red-400"
              }`}
            >
              {c.status}
            </span>
            <span className="font-medium">{c.name}</span>
            {c.detail && <span className="text-sm text-zinc-500">{c.detail}</span>}
          </li>
        ))}
      </ul>
    </div>
  );
}
