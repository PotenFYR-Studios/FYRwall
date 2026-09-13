import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { motion, useReducedMotion } from "framer-motion";
import { Activity, Copy, Radio, Server, ShieldAlert, ShieldCheck } from "lucide-react";
import { NumberTicker } from "../components/magicui";
import { useState } from "react";

import { fetchJSON, sendJSON } from "../lib/auth";

type Agent = {
  agent_id: string;
  display_name: string;
  hostname: string;
  architecture: string;
  distribution: string;
  agent_version: string;
  firewall_backend: string;
  firewall_owner: string;
  connection_state: string;
  health_state: string;
  last_seen_at?: string;
  sequence: number;
  state_hash?: string;
  policy_revision: number;
  labels?: string[];
  groups?: string[];
  site?: string;
};

type FleetNotification = {
  code: string;
  component: string;
};

function healthRank(agent: Agent): number {
  if (agent.health_state === "DEGRADED") return 0;
  if (agent.health_state === "UNKNOWN") return 1;
  return 2;
}

function healthLabel(agent: Agent): string {
  if (agent.health_state === "HEALTHY") return "Healthy";
  if (agent.health_state === "DEGRADED") return "Degraded";
  return "Unknown";
}

function healthStyles(agent: Agent): string {
  if (agent.health_state === "DEGRADED") return "bg-red-500/15 text-red-300";
  if (agent.health_state === "UNKNOWN") return "bg-zinc-500/15 text-zinc-300";
  return "bg-emerald-500/15 text-emerald-300";
}

export default function Fleet() {
  const qc = useQueryClient();
  const [token, setToken] = useState("");
  const agents = useQuery({
    queryKey: ["agents"],
    queryFn: () => fetchJSON<{ agents: Agent[] }>("/api/v1/agents"),
    refetchInterval: 5_000,
  });
  const drift = useQuery({
    queryKey: ["notifications", "fleet-drift"],
    queryFn: () => fetchJSON<{ notifications: FleetNotification[] }>("/api/v1/notifications"),
    refetchInterval: 5_000,
    select: (data) => new Set(
      data.notifications
        .filter((notification) => notification.code === "AGENT_FIREWALL_DRIFT")
        .map((notification) => notification.component.replace(/^agent:/, "")),
    ),
  });
  const allAgents = agents.data?.agents ?? [];
  const agentList = [...allAgents].sort((left, right) => healthRank(left) - healthRank(right));
  const driftAgents = drift.data ?? new Set<string>();
  const onlineCount = allAgents.filter((agent) => {
    if (agent.connection_state !== "ONLINE" || !agent.last_seen_at) return false;
    return Date.now() - new Date(agent.last_seen_at).getTime() < 45_000;
  }).length;
  const degradedCount = allAgents.filter((agent) => agent.health_state === "DEGRADED").length;
  const driftCount = drift.data?.size ?? 0;
  const reducedMotion = useReducedMotion();
  const needsAttention = (agent: Agent) => driftAgents.has(agent.agent_id) || agent.health_state === "DEGRADED";
  const createToken = useMutation({
    mutationFn: () => sendJSON<{ token: string }>("/api/v1/agents/enrollment-tokens", "POST", { ttl_minutes: 15 }),
    onSuccess: (data) => setToken(data.token),
  });
  const refresh = useMutation({
    mutationFn: (agentID: string) => sendJSON(`/api/v1/agents/${agentID}/commands`, "POST", { op: "GetFirewallStatus" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["agents"] }),
  });
  const revoke = useMutation({
    mutationFn: (agentID: string) => sendJSON(`/api/v1/agents/${agentID}`, "DELETE"),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["agents"] }),
  });

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.24em] text-amber-400">Control plane</p>
          <h1 className="mt-1 text-3xl font-bold tracking-tight">Managed targets</h1>
          <p className="mt-2 max-w-2xl text-sm text-zinc-400">Outbound agents synchronize firewall state, drift hashes, health, inventory, and policy revisions.</p>
        </div>
        <button onClick={() => createToken.mutate()} className="rounded bg-amber-400 px-4 py-2 text-sm font-semibold text-zinc-950 hover:bg-amber-300">Create 15-minute enrollment token</button>
      </div>

      {token && <div className="flex items-center gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 p-4"><code className="min-w-0 flex-1 break-all text-sm text-amber-200">{token}</code><button title="Copy token" onClick={() => navigator.clipboard.writeText(token)}><Copy className="h-4 w-4" /></button></div>}
      {createToken.error && <p className="text-sm text-red-400">{(createToken.error as Error).message}</p>}

      <div className="grid grid-cols-2 gap-4 lg:grid-cols-3">
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/70 p-4">
          <p className="text-xs uppercase tracking-wide text-zinc-500">Online</p>
          <p className="mt-1 flex items-baseline gap-2 text-2xl font-semibold text-white">
            <NumberTicker value={onlineCount} />
            <span className="text-sm text-zinc-500">/ {allAgents.length}</span>
          </p>
        </div>
        <div className={`rounded-xl border p-4 transition ${driftCount ? "border-amber-500/40 bg-amber-500/10" : "border-zinc-800 bg-zinc-900/70"}`}>
          <p className={`text-xs uppercase tracking-wide ${driftCount ? "text-amber-300" : "text-zinc-500"}`}>Drift</p>
          <p className={`mt-1 text-2xl font-semibold ${driftCount ? "text-amber-200" : "text-zinc-200"}`}><NumberTicker value={driftCount} /></p>
        </div>
        <div className="col-span-2 rounded-xl border border-zinc-800 bg-zinc-900/70 p-4 lg:col-span-1">
          <p className="text-xs uppercase tracking-wide text-zinc-500">Health</p>
          <p className="mt-1 flex items-baseline gap-2 text-2xl font-semibold text-white">
            <NumberTicker value={degradedCount} />
            <span className="text-sm text-zinc-500">degraded</span>
          </p>
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        {agentList.map((agent, index) => {
          const online = agent.connection_state === "ONLINE" && !!agent.last_seen_at && Date.now() - new Date(agent.last_seen_at).getTime() < 45_000;
          return (
            <motion.article
              key={agent.agent_id}
              initial={reducedMotion ? { opacity: 0 } : { opacity: 0, y: 18 }}
              animate={reducedMotion ? { opacity: 1 } : { opacity: 1, y: 0 }}
              transition={reducedMotion ? { delay: index * 0.025, duration: 0.2 } : { delay: index * 0.055, duration: 0.4 }}
              className={`relative overflow-hidden rounded-xl border bg-gradient-to-br from-zinc-900 to-zinc-950 p-5 ${needsAttention(agent) ? "border-amber-500/40" : "border-zinc-800"}`}
            >
              <div className="flex items-start justify-between gap-4">
                <div className="flex gap-3">
                  <div className="rounded-lg bg-zinc-800 p-2"><Server className="h-5 w-5 text-amber-400" /></div>
                  <div>
                    <h2 className="font-semibold">{agent.display_name || agent.hostname}</h2>
                    <p className="text-xs text-zinc-500">{agent.hostname} · {agent.agent_id.slice(0, 8)}</p>
                  </div>
                </div>
                <div className="flex shrink-0 flex-col items-end gap-2">
                  <span className={`flex items-center gap-1 rounded-full px-2 py-0.5 text-xs ${online ? "bg-emerald-500/15 text-emerald-300" : "bg-zinc-500/15 text-zinc-300"}`}>
                    <Radio className={`h-3 w-3 ${online ? "text-emerald-400" : "text-zinc-500"}`} />
                    {online ? "Online" : "Offline"}
                  </span>
                  <span className={`flex items-center gap-1 rounded-full px-2 py-0.5 text-xs ${healthStyles(agent)}`}>
                    {agent.health_state === "DEGRADED" ? <ShieldAlert className="h-3 w-3" /> : <ShieldCheck className="h-3 w-3" />}
                    {healthLabel(agent)}
                  </span>
                </div>
              </div>
              <div className="mt-5 grid grid-cols-2 gap-3 text-sm">
                <Metric icon={ShieldCheck} label="Firewall" value={agent.firewall_owner || agent.firewall_backend || "Unknown"} />
                <Metric icon={Activity} label="Health" value={agent.health_state} />
                <Metric label="Platform" value={`${agent.distribution || "Linux"} / ${agent.architecture}`} />
                <Metric label="Policy revision" value={String(agent.policy_revision)} />
              </div>
            {!!agent.groups?.length && <div className="mt-3 flex flex-wrap gap-1">{agent.groups.map((group) => <span key={group} className="rounded bg-zinc-800 px-2 py-1 text-[10px] uppercase tracking-wide text-zinc-400">{group}</span>)}</div>}
            <div className="mt-4 flex items-center justify-between border-t border-zinc-800 pt-4"><span className="text-xs text-zinc-500">Seen {agent.last_seen_at ? new Date(agent.last_seen_at).toLocaleString() : "never"}</span><div className="flex gap-3"><button disabled={!online || refresh.isPending} onClick={() => refresh.mutate(agent.agent_id)} className="text-xs text-amber-400 disabled:text-zinc-700">Request status</button><button onClick={() => { if (confirm(`Revoke ${agent.display_name || agent.hostname}?`)) revoke.mutate(agent.agent_id); }} className="text-xs text-red-400">Revoke</button></div></div>
            </motion.article>
          );
        })}
      </div>
      {!agents.isLoading && !agents.data?.agents.length && <div className="rounded-xl border border-dashed border-zinc-800 p-12 text-center text-zinc-500">No agents enrolled. Create token, then start target with <code className="text-zinc-300">fyrwall agent --server https://your-server</code>.</div>}
    </div>
  );
}

function Metric({ label, value, icon: Icon }: { label: string; value: string; icon?: React.ComponentType<{ className?: string }> }) { return <div className="rounded-lg bg-zinc-900/80 p-3"><div className="flex items-center gap-1 text-[11px] uppercase tracking-wide text-zinc-600">{Icon && <Icon className="h-3 w-3" />}{label}</div><div className="mt-1 truncate text-zinc-200">{value}</div></div>; }
