"use client";

import { useEffect, useRef, useState } from "react";
import { ScrollText, Radio } from "lucide-react";
import { Badge, Button, H1, Label, Pagination, Panel, Select, TableWrap } from "@/components/ui";
import { api, AuditEntry, User } from "@/lib/api";
import { formatDateTime } from "@/lib/time";
import { useRealtime } from "@/lib/ws";

const PAGE_SIZE = 50;

// Every distinct audit action string the backend actually writes (kept in
// sync with the Action: "..." literals in internal/httpserver) -- shown as
// a dropdown instead of asking an admin to remember/type them.
const ACTION_OPTIONS = [
  "user.login",
  "user.create",
  "user.update",
  "user.delete",
  "user.offboard",
  "user.reset_password",
  "user.avatar_upload",
  "device_credential.create",
  "api_client.create",
  "iot_device.create",
];

export default function AuditPage() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [offset, setOffset] = useState(0);
  const [users, setUsers] = useState<User[]>([]);
  const [actorId, setActorId] = useState("");
  const [action, setAction] = useState("");
  const [status, setStatus] = useState("");
  const filtered = useRef(false);
  const liveIdCounter = useRef(-1);

  useEffect(() => {
    api.users.list(500, 0).then((r) => setUsers(r.users ?? []));
  }, []);

  function load(atOffset: number) {
    filtered.current = Boolean(actorId || action || status);
    const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(atOffset) });
    if (actorId) params.set("actor_id", actorId);
    if (action) params.set("action", action);
    if (status) params.set("status", status);
    api.auditLogs(`?${params.toString()}`).then((r) => setEntries(r.entries ?? []));
  }

  function applyFilter() {
    setOffset(0);
    load(0);
  }

  useEffect(() => load(offset), [offset]); // eslint-disable-line react-hooks/exhaustive-deps

  // Live-prepend new entries as they happen, as long as no filter narrows
  // the view -- a filtered view would otherwise show entries that don't
  // match the filter.
  useRealtime((e) => {
    if (e.type !== "audit_log" || filtered.current || offset !== 0) return;
    setEntries((prev) =>
      [
        {
          id: liveIdCounter.current--,
          actor_id: e.actor_id,
          action: e.action,
          target_resource: e.target_resource,
          status: e.status,
          timestamp: e.timestamp,
        },
        ...prev,
      ].slice(0, PAGE_SIZE)
    );
  });

  return (
    <>
      <div className="flex items-center justify-between mb-1">
        <H1 icon={<ScrollText size={20} />}>Audit log</H1>
        <span className="flex items-center gap-1.5 text-xs text-ok font-medium">
          <Radio size={13} className="animate-pulse" />
          Live
        </span>
      </div>

      <Panel>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div>
            <Label>Actor</Label>
            <Select value={actorId} onChange={(e) => setActorId(e.target.value)}>
              <option value="">All actors</option>
              {users.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.username}
                </option>
              ))}
            </Select>
          </div>
          <div>
            <Label>Action</Label>
            <Select value={action} onChange={(e) => setAction(e.target.value)}>
              <option value="">All actions</option>
              {ACTION_OPTIONS.map((a) => (
                <option key={a} value={a}>
                  {a}
                </option>
              ))}
            </Select>
          </div>
          <div>
            <Label>Status</Label>
            <Select value={status} onChange={(e) => setStatus(e.target.value)}>
              <option value="">All statuses</option>
              <option value="success">success</option>
              <option value="failure">failure</option>
            </Select>
          </div>
        </div>
        <p className="text-muted text-xs mt-2">
          Actor list only covers user accounts. To filter by an API client or device, check its ID
          on the API clients / IoT devices page and search this log&apos;s entries visually for now.
        </p>
        <Button className="mt-3" onClick={applyFilter}>
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
                <td className="py-2.5 px-3 font-mono text-xs whitespace-nowrap">{formatDateTime(e.timestamp)}</td>
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
      <Pagination
        offset={offset}
        limit={PAGE_SIZE}
        count={entries.length}
        onPrev={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
        onNext={() => setOffset((o) => o + PAGE_SIZE)}
      />
    </>
  );
}
