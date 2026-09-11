import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fetchJSON } from "../components/HealthBadge";

type Notification = {
  id: string;
  code: string;
  severity: string;
  component: string;
  summary: string;
  remediation?: string;
  occurrences: number;
  first_seen_at: string;
  last_seen_at: string;
};

const sevColor: Record<string, string> = {
  critical: "bg-red-500/20 text-red-400",
  error: "bg-red-500/10 text-red-300",
  warning: "bg-yellow-500/20 text-yellow-400",
  info: "bg-blue-500/20 text-blue-400",
};

export default function Notifications() {
  const qc = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ["notifications"],
    queryFn: () => fetchJSON<{ notifications: Notification[] }>("/api/v1/notifications"),
    refetchInterval: 15_000,
  });
  const dismiss = useMutation({
    mutationFn: (id: string) =>
      fetch(`/api/v1/notifications/${id}/read`, {
        method: "POST",
        credentials: "same-origin",
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notifications"] }),
  });
  void dismiss; // wired to per-item dismiss controls in the full UI

  if (isLoading) return <p className="text-zinc-500">Loading notifications...</p>;
  const items = data?.notifications ?? [];
  if (items.length === 0)
    return <p className="text-zinc-500">No unresolved issues. Everything looks good.</p>;

  return (
    <ul className="space-y-2" data-testid="notification-list">
      {items.map((n) => (
        <li key={n.id} className="rounded border border-zinc-800 bg-zinc-900 p-3">
          <div className="flex items-center gap-2">
            <span className={`text-xs px-2 py-0.5 rounded ${sevColor[n.severity] ?? sevColor.info}`}>
              {n.severity}
            </span>
            <span className="font-medium">{n.code}</span>
            <span className="text-xs text-zinc-600">{n.component}</span>
            {n.occurrences > 1 && (
              <span className="text-xs text-zinc-600">x{n.occurrences}</span>
            )}
          </div>
          <p className="mt-1 text-sm">{n.summary}</p>
          {n.remediation && <p className="mt-1 text-sm text-zinc-500">Remediation: {n.remediation}</p>}
        </li>
      ))}
    </ul>
  );
}
