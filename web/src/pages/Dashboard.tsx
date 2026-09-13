import { useQuery } from "@tanstack/react-query";
import { Activity, ShieldAlert, ShieldCheck, Waypoints } from "lucide-react";

import { fetchJSON } from "../components/HealthBadge";
import { BorderBeam, NumberTicker } from "../components/magicui";

type FirewallStatus = {
  status: {
    backend: string;
    enabled: boolean;
    rule_count: number;
    ipv4_ready: boolean;
    ipv6_ready: boolean;
    default_policies: { direction: string; action: string }[];
  };
  ownership: { owner: string; writes_allowed: boolean; reason?: string };
};

function Card({
  title,
  value,
  warn,
  ticker,
  icon: Icon,
}: {
  title: string;
  value: string | number;
  warn?: boolean;
  ticker?: boolean;
  icon?: React.ComponentType<{ className?: string }>;
}) {
  return (
    <div className="relative overflow-hidden rounded-lg border border-zinc-800 bg-zinc-900 p-4 transition hover:border-zinc-700">
      {ticker && <BorderBeam size={40} duration={9} />}
      <div className="flex items-center justify-between">
        <div className="text-xs uppercase tracking-wide text-zinc-500">{title}</div>
        {Icon && <Icon className="h-4 w-4 text-zinc-600" />}
      </div>
      <div className={`mt-1 text-xl font-semibold ${warn ? "text-yellow-400" : "text-zinc-100"}`}>
        {ticker && typeof value === "number" ? (
          <NumberTicker value={value} />
        ) : (
          value
        )}
      </div>
    </div>
  );
}

export default function Dashboard() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["firewall-status"],
    queryFn: () => fetchJSON<FirewallStatus>("/api/v1/firewall/status"),
    refetchInterval: 20_000,
  });

  if (isLoading) return <p className="text-zinc-500">Loading firewall status...</p>;
  if (error)
    return (
      <div className="rounded-lg border border-red-900 bg-red-950/40 p-4 text-red-300">
        Firewall backend unavailable: {(error as Error).message}
      </div>
    );

  const s = data!.status;
  const own = data!.ownership;
  return (
    <div className="space-y-4">
      {own.owner === "MULTIPLE_CONFLICTING" && (
        <div className="flex items-start gap-2 rounded-lg border border-red-900 bg-red-950/40 p-4" data-testid="conflict-banner">
          <ShieldAlert className="mt-0.5 h-5 w-5 shrink-0 text-red-400" />
          <div>
            <p className="font-semibold text-red-300">Multiple firewall managers detected</p>
            <p className="mt-1 text-sm text-red-400">{own.reason}</p>
          </div>
        </div>
      )}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card title="Backend" value={s.backend} icon={Waypoints} />
        <Card title="Enabled" value={s.enabled ? "Yes" : "No"} warn={!s.enabled} icon={ShieldCheck} />
        <Card title="Active rules" value={s.rule_count} ticker icon={Activity} />
        <Card title="Policy owner" value={own.owner} warn={!own.writes_allowed} />
        <Card title="Inbound default" value={s.default_policies.find((p) => p.direction === "in")?.action ?? "unknown"} />
        <Card title="Outbound default" value={s.default_policies.find((p) => p.direction === "out")?.action ?? "unknown"} />
        <Card title="IPv4" value={s.ipv4_ready ? "Ready" : "Unavailable"} warn={!s.ipv4_ready} />
        <Card title="IPv6" value={s.ipv6_ready ? "Ready" : "Unavailable"} />
      </div>
      {!own.writes_allowed && (
        <p className="text-sm text-zinc-500">Firewall writes are blocked: {own.reason}</p>
      )}
    </div>
  );
}
