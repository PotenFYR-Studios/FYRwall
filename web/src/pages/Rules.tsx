import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, RotateCcw, Trash2 } from "lucide-react";
import { FormEvent, useState } from "react";

import { fetchJSON, sendJSON } from "../lib/auth";

type Rule = { id:string; backend_id?:string; family:string; direction:string; action:string; protocol:string; source:string; destination:string; destination_port:string; enabled:boolean; comment:string };
type Agent = { agent_id:string; display_name:string; hostname:string; groups?:string[] };
type Command = { command_id:string; state:string; error?:string };
type Rollout = { rollout_id:string; state:string; total_targets:number; completed_targets:number; failed_targets:number };

export default function Rules() {
  const qc=useQueryClient();
  const [target,setTarget]=useState("local");
  const [commandID,setCommandID]=useState("");
  const [rolloutID,setRolloutID]=useState("");
  const [form,setForm]=useState({action:"allow",direction:"in",protocol:"tcp",source:"any",destination:"any",port:"",comment:""});
  const agents=useQuery({queryKey:["agents"],queryFn:()=>fetchJSON<{agents:Agent[]}>("/api/v1/agents"),refetchInterval:10_000});
  const groupTarget=target.startsWith("group:");
  const remote=target!=="local"&&!groupTarget;
  const rulesURL=remote?`/api/v1/agents/${target}/firewall/rules`:"/api/v1/firewall/rules";
  const rules=useQuery({queryKey:["rules",target],queryFn:()=>fetchJSON<{rules:Rule[]}>(rulesURL),enabled:!groupTarget,refetchInterval:remote?5_000:15_000});
  const command=useQuery({queryKey:["agent-command",commandID],queryFn:()=>fetchJSON<Command>(`/api/v1/agent-commands/${commandID}`),enabled:!!commandID,refetchInterval:q=>["COMPLETED","FAILED"].includes(q.state.data?.state??"")?false:800});
  const rollout=useQuery({queryKey:["rollout",rolloutID],queryFn:()=>fetchJSON<Rollout>(`/api/v1/rollouts/${rolloutID}`),enabled:!!rolloutID,refetchInterval:q=>["COMPLETED","FAILED"].includes(q.state.data?.state??"")?false:800});
  const apply=useMutation<Command|Rollout|null,Error,unknown>({
    mutationFn:async(tx:unknown)=>{
      if(groupTarget)return sendJSON<Rollout>(`/api/v1/agent-groups/${encodeURIComponent(target.slice(6))}/rollouts`,"POST",{op:"ApplyTransaction",params:tx,batch_size:1,max_failures:0});
      if(remote)return sendJSON<Command>(`/api/v1/agents/${target}/firewall/transactions`,"POST",tx);
      await sendJSON("/api/v1/firewall/transactions","POST",tx);return null;
    },
    onSuccess:result=>{if(result&&"rollout_id" in result)setRolloutID(result.rollout_id);else if(result)setCommandID(result.command_id);qc.invalidateQueries({queryKey:["rules",target]})},
  });
  const emergency=useMutation<Command|Rollout,Error,void>({
    mutationFn:()=>groupTarget
      ?sendJSON<Rollout>(`/api/v1/agent-groups/${encodeURIComponent(target.slice(6))}/rollouts`,"POST",{op:"RestoreLast",batch_size:1,max_failures:0})
      :sendJSON<Command>(`/api/v1/agents/${target}/commands`,"POST",{op:"RestoreLast"}),
    onSuccess:result=>{"rollout_id" in result?setRolloutID(result.rollout_id):setCommandID(result.command_id)},
  });
  function tx(op:string,rule:Rule|Record<string,unknown>){return{id:crypto.randomUUID(),backend:"auto",created_at:new Date().toISOString(),actions:[{op,rule,rule_id:(rule as Rule).backend_id||(rule as Rule).id}]}}
  function add(e:FormEvent){e.preventDefault();apply.mutate(tx("add",{id:crypto.randomUUID(),family:"ipv4",direction:form.direction,action:form.action,protocol:form.protocol,source:form.source,destination:form.destination,destination_port:form.port,enabled:true,comment:form.comment,priority:0,backend:""}))}
  const groups=[...new Set((agents.data?.agents??[]).flatMap(a=>a.groups??[]))].sort();
  const rows=rules.data?.rules??[];

  return <div className="space-y-5">
    <div className="flex flex-wrap items-end justify-between gap-4">
      <div><p className="text-xs font-semibold uppercase tracking-[0.22em] text-amber-400">Policy workspace</p><h1 className="mt-1 text-3xl font-bold">Firewall rules</h1></div>
      <label className="text-xs text-zinc-500">Target <select value={target} onChange={e=>{setTarget(e.target.value);setCommandID("");setRolloutID("")}} className="ml-2 rounded border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-white"><option value="local">Local server host</option><optgroup label="Agents">{(agents.data?.agents??[]).map(a=><option key={a.agent_id} value={a.agent_id}>{a.display_name||a.hostname}</option>)}</optgroup>{groups.length>0&&<optgroup label="Staged groups">{groups.map(g=><option key={g} value={`group:${g}`}>{g}</option>)}</optgroup>}</select></label>
    </div>
    {commandID&&<Progress failed={command.data?.state==="FAILED"}>Command {commandID.slice(0,8)}: {command.data?.state??"QUEUED"}{command.data?.error&&` - ${command.data.error}`}</Progress>}
    {rolloutID&&<Progress failed={rollout.data?.state==="FAILED"}>Rollout {rolloutID.slice(0,8)}: {rollout.data?.state??"RUNNING"} ({rollout.data?.completed_targets??0}/{rollout.data?.total_targets??0}, {rollout.data?.failed_targets??0} failed)</Progress>}
    <form onSubmit={add} className="grid gap-3 rounded-xl border border-zinc-800 bg-zinc-900/70 p-4 md:grid-cols-4">
      <Select label="Action" value={form.action} values={["allow","deny","reject","drop"]} onChange={v=>setForm({...form,action:v})}/><Select label="Direction" value={form.direction} values={["in","out","forward"]} onChange={v=>setForm({...form,direction:v})}/><Select label="Protocol" value={form.protocol} values={["tcp","udp","icmp","any"]} onChange={v=>setForm({...form,protocol:v})}/><Field label="Port" value={form.port} onChange={v=>setForm({...form,port:v})} placeholder="22 or 8000:8080"/><Field label="Source" value={form.source} onChange={v=>setForm({...form,source:v})}/><Field label="Destination" value={form.destination} onChange={v=>setForm({...form,destination:v})}/><div className="md:col-span-2"><Field label="Comment" value={form.comment} onChange={v=>setForm({...form,comment:v})}/></div>
      <button disabled={apply.isPending} className="flex items-center justify-center gap-2 rounded bg-amber-400 px-4 py-2 font-semibold text-zinc-950 hover:bg-amber-300 disabled:opacity-50"><Plus className="h-4 w-4"/>Add rule</button>
      {target!=="local"&&<button type="button" onClick={()=>emergency.mutate()} className="flex items-center justify-center gap-2 rounded border border-red-800 px-4 py-2 text-red-300 hover:bg-red-950/40"><RotateCcw className="h-4 w-4"/>Emergency rollback</button>}
    </form>
    {apply.error&&<p className="text-sm text-red-400">{(apply.error as Error).message}</p>}
    {groupTarget?<p className="rounded-xl border border-dashed border-zinc-800 p-8 text-center text-zinc-500">Group target applies staged batches. Select individual agent to inspect current rules.</p>:rules.isLoading?<p className="text-zinc-500">Loading rules...</p>:rules.error?<p className="text-red-400">Failed to load rules: {(rules.error as Error).message}</p>:<RuleTable rows={rows} remove={rule=>apply.mutate(tx("delete",rule))}/>}
  </div>;
}

function Progress({failed,children}:{failed:boolean;children:React.ReactNode}){return <div className={`rounded-lg border p-3 text-sm ${failed?"border-red-800 bg-red-950/30 text-red-300":"border-amber-700/40 bg-amber-500/10 text-amber-200"}`}>{children}</div>}
function RuleTable({rows,remove}:{rows:Rule[];remove:(r:Rule)=>void}){return <div className="overflow-x-auto rounded-xl border border-zinc-800"><table className="w-full text-sm" data-testid="rules-table"><thead><tr className="border-b border-zinc-800 bg-zinc-900 text-left text-zinc-500"><th className="p-3">Action</th><th>Direction</th><th>Proto</th><th>Source</th><th>Destination</th><th>Port</th><th>Family</th><th>Comment</th><th/></tr></thead><tbody>{rows.map(r=><tr key={r.id} className="border-b border-zinc-900 hover:bg-zinc-900/50"><td className={`p-3 font-medium ${r.action==="allow"?"text-green-400":"text-red-400"}`}>{r.action.toUpperCase()}</td><td>{r.direction}</td><td>{r.protocol}</td><td>{r.source||"any"}</td><td>{r.destination||"any"}</td><td>{r.destination_port||"-"}</td><td>{r.family}</td><td className="text-zinc-500">{r.comment}</td><td><button title="Delete rule" onClick={()=>remove(r)} className="p-2 text-zinc-600 hover:text-red-400"><Trash2 className="h-4 w-4"/></button></td></tr>)}</tbody></table>{rows.length===0&&<p className="p-8 text-center text-zinc-500">No rules found.</p>}</div>}
function Field({label,value,onChange,placeholder}:{label:string;value:string;onChange:(v:string)=>void;placeholder?:string}){return <label className="block text-xs text-zinc-500">{label}<input value={value} placeholder={placeholder} onChange={e=>onChange(e.target.value)} className="mt-1 w-full rounded border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-white"/></label>}
function Select({label,value,values,onChange}:{label:string;value:string;values:string[];onChange:(v:string)=>void}){return <label className="block text-xs text-zinc-500">{label}<select value={value} onChange={e=>onChange(e.target.value)} className="mt-1 w-full rounded border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-white">{values.map(v=><option key={v}>{v}</option>)}</select></label>}
