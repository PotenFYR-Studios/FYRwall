import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";

import { fetchJSON, sendJSON } from "../lib/auth";
import { AnimatedList } from "../components/magicui";

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
      sendJSON(`/api/v1/notifications/${id}/read`, "POST"),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notifications"] }),
  });

  if (isLoading) return <p className="text-zinc-500">Loading notifications...</p>;
  const items = data?.notifications ?? [];
  if (items.length === 0)
    return <p className="text-zinc-500">No unresolved issues. Everything looks good.</p>;

  return (
    <AnimatedList className="max-w-3xl" data-testid="notification-list">
      {items.map((n) => (
        <li
          key={n.id}
          className="relative overflow-hidden rounded border border-zinc-800 bg-zinc-900 p-3"
        >
          <div className="flex items-center gap-2">
            <span className={`text-xs px-2 py-0.5 rounded ${sevColor[n.severity] ?? sevColor.info}`}>
              {n.severity}
            </span>
            <span className="font-medium">{n.code}</span>
            <span className="text-xs text-zinc-600">{n.component}</span>
            {n.occurrences > 1 && (
              <span className="text-xs text-zinc-600">x{n.occurrences}</span>
            )}
            <button
              onClick={() => dismiss.mutate(n.id)}
              disabled={dismiss.isPending}
              className="ml-auto rounded border border-zinc-800 px-2 py-0.5 text-xs text-zinc-400 transition hover:text-white"
            >
              Dismiss
            </button>
          </div>
          <p className="mt-1 text-sm">{n.summary}</p>
          {n.remediation && <p className="mt-1 text-sm text-zinc-500">Remediation: {n.remediation}</p>}
        </li>
      ))}
    </AnimatedList>
  );
}
