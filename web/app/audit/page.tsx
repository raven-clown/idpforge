"use client";

import { useEffect, useState } from "react";
import { ScrollText } from "lucide-react";
import AppShell from "@/components/AppShell";
import { Badge, Button, H1, Input, Label, Panel, TableWrap } from "@/components/ui";
import { api, AuditEntry } from "@/lib/api";

export default function AuditPage() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [actorId, setActorId] = useState("");
  const [action, setAction] = useState("");
  const [status, setStatus] = useState("");

  function refresh() {
    const params = new URLSearchParams({ limit: "200" });
    if (actorId) params.set("actor_id", actorId);
    if (action) params.set("action", action);
    if (status) params.set("status", status);
    api.auditLogs(`?${params.toString()}`).then((r) => setEntries(r.entries ?? []));
  }

  useEffect(refresh, []); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <AppShell>
      <H1 icon={<ScrollText size={20} />}>Audit log</H1>

      <Panel>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div>
            <Label>Actor ID</Label>
            <Input value={actorId} onChange={(e) => setActorId(e.target.value)} placeholder="user or apiclient:id" />
          </div>
          <div>
            <Label>Action</Label>
            <Input value={action} onChange={(e) => setAction(e.target.value)} placeholder="user.login, user.create, ..." />
          </div>
          <div>
            <Label>Status</Label>
            <Input value={status} onChange={(e) => setStatus(e.target.value)} placeholder="success, failure" />
          </div>
        </div>
        <Button className="mt-4" onClick={refresh}>
          Filter
        </Button>
      </Panel>

      <TableWrap>
        <table className="w-full border-collapse min-w-[640px]">
          <thead>
            <tr className="text-left text-[11.5px] uppercase tracking-wide text-muted">
              <th className="py-2.5 px-3">Time</th>
              <th className="py-2.5 px-3">Actor</th>
              <th className="py-2.5 px-3">Action</th>
              <th className="py-2.5 px-3">Target</th>
              <th className="py-2.5 px-3">IP</th>
              <th className="py-2.5 px-3">Status</th>
            </tr>
          </thead>
          <tbody>
            {entries.map((e) => (
              <tr key={e.id} className="border-t border-border hover:bg-accent-soft transition-colors">
                <td className="py-2.5 px-3 font-mono text-xs whitespace-nowrap">
                  {new Date(e.timestamp).toISOString().slice(0, 19).replace("T", " ")} UTC
                </td>
                <td className="py-2.5 px-3 font-mono text-xs">{e.actor_id}</td>
                <td className="py-2.5 px-3">{e.action}</td>
                <td className="py-2.5 px-3 text-muted">{e.target_resource || e.target_app}</td>
                <td className="py-2.5 px-3 text-muted">{e.actor_ip}</td>
                <td className="py-2.5 px-3">
                  <Badge tone={e.status === "success" ? "ok" : "danger"}>{e.status}</Badge>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </TableWrap>
    </AppShell>
  );
}
