import { useQuery } from "@tanstack/react-query";
import { fetchJSON } from "../components/HealthBadge";

type Rule = {
  id: string;
  family: string;
  direction: string;
  action: string;
  protocol: string;
  source: string;
  destination: string;
  destination_port: string;
  enabled: boolean;
  comment: string;
};

export default function Rules() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["rules"],
    queryFn: () => fetchJSON<{ rules: Rule[]; count: number }>("/api/v1/firewall/rules"),
  });

  if (isLoading) return <p className="text-zinc-500">Loading rules...</p>;
  if (error) return <p className="text-red-400">Failed to load rules: {(error as Error).message}</p>;

  const rules = data!.rules ?? [];
  if (rules.length === 0)
    return <p className="text-zinc-500">No rules found. The firewall may be empty or unreadable.</p>;

  return (
    <div>
      <h2 className="text-lg font-semibold mb-3">Rules ({data!.count})</h2>
      <table className="w-full text-sm" data-testid="rules-table">
        <thead>
          <tr className="text-left text-zinc-500 border-b border-zinc-800">
            <th className="py-2">Action</th>
            <th>Direction</th>
            <th>Proto</th>
            <th>Source</th>
            <th>Destination</th>
            <th>Port</th>
            <th>Family</th>
            <th>Comment</th>
          </tr>
        </thead>
        <tbody>
          {rules.map((r) => (
            <tr key={r.id} className="border-b border-zinc-900 hover:bg-zinc-900/50">
              <td className={`py-2 font-medium ${r.action === "allow" ? "text-green-400" : "text-red-400"}`}>
                {r.action.toUpperCase()}
              </td>
              <td>{r.direction}</td>
              <td>{r.protocol}</td>
              <td>{r.source || "any"}</td>
              <td>{r.destination || "any"}</td>
              <td>{r.destination_port || "-"}</td>
              <td>{r.family}</td>
              <td className="text-zinc-500">{r.comment}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
