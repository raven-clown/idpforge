"use client";

import { useEffect, useState } from "react";
import { KeyRound } from "lucide-react";
import AppShell from "@/components/AppShell";
import { Button, Flash, H1, Input, Label, Panel, TableWrap, Textarea } from "@/components/ui";
import { api, ApiClient, ApiError } from "@/lib/api";

function splitCSV(s: string): string[] {
  return s.split(",").map((v) => v.trim()).filter(Boolean);
}
function splitLines(s: string): string[] {
  return s.split("\n").map((v) => v.trim()).filter(Boolean);
}

export default function ApiClientsPage() {
  const [clients, setClients] = useState<ApiClient[]>([]);
  const [name, setName] = useState("");
  const [scopes, setScopes] = useState("");
  const [allowedFields, setAllowedFields] = useState("");
  const [allowedIps, setAllowedIps] = useState("");
  const [rateLimitMax, setRateLimitMax] = useState(60);
  const [rateLimitWindow, setRateLimitWindow] = useState(60);
  const [newKey, setNewKey] = useState("");
  const [flash, setFlash] = useState<{ msg: string; kind: "ok" | "error" } | null>(null);

  function refresh() {
    api.apiClients.list().then((r) => setClients(r.clients ?? []));
  }
  useEffect(refresh, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      const { api_key } = await api.apiClients.create({
        name,
        scopes: splitLines(scopes),
        allowed_fields: splitCSV(allowedFields),
        allowed_ips: splitCSV(allowedIps),
        rate_limit_max: rateLimitMax,
        rate_limit_window_seconds: rateLimitWindow,
      });
      setNewKey(api_key);
      setFlash({ msg: "API client created", kind: "ok" });
      setName("");
      setScopes("");
      setAllowedFields("");
      setAllowedIps("");
      refresh();
    } catch (err) {
      setFlash({ msg: err instanceof ApiError ? err.message : "Could not create API client", kind: "error" });
    }
  }

  return (
    <AppShell>
      <H1 icon={<KeyRound size={20} />}>API clients</H1>
      {flash && <Flash message={flash.msg} kind={flash.kind} />}

      {newKey && (
        <Panel>
          <strong className="text-sm">New API key created. Copy it now, it cannot be shown again:</strong>
          <div className="bg-input-bg border border-dashed border-accent rounded-lg p-3.5 mt-3 font-mono text-[13px] break-all">
            {newKey}
          </div>
        </Panel>
      )}

      <Panel>
        <h2 className="text-sm font-semibold mb-1">New API client</h2>
        <form onSubmit={handleCreate}>
          <Label>Name</Label>
          <Input value={name} onChange={(e) => setName(e.target.value)} required />

          <Label>Scopes (for /api/v1, resource:action per line, e.g. users:read)</Label>
          <Textarea rows={3} value={scopes} onChange={(e) => setScopes(e.target.value)} placeholder={"users:read\niot:read"} />

          <Label>Allowed fields (for /external/v1 read-only path, comma-separated)</Label>
          <Input value={allowedFields} onChange={(e) => setAllowedFields(e.target.value)} placeholder="id,username,email" />

          <Label>Allowed IPs / CIDRs (comma-separated, blank = unrestricted)</Label>
          <Input value={allowedIps} onChange={(e) => setAllowedIps(e.target.value)} placeholder="10.0.0.0/24" />

          <Label>Rate limit (requests per window)</Label>
          <Input type="number" value={rateLimitMax} onChange={(e) => setRateLimitMax(Number(e.target.value))} />
          <Label>Rate limit window (seconds)</Label>
          <Input type="number" value={rateLimitWindow} onChange={(e) => setRateLimitWindow(Number(e.target.value))} />

          <Button type="submit" className="mt-4">
            Create
          </Button>
        </form>
      </Panel>

      <TableWrap>
        <table className="w-full border-collapse min-w-[560px]">
          <thead>
            <tr className="text-left text-[11.5px] uppercase tracking-wide text-muted">
              <th className="py-2.5 px-3">Name</th>
              <th className="py-2.5 px-3">Scopes</th>
              <th className="py-2.5 px-3">Allowed fields</th>
              <th className="py-2.5 px-3">Rate limit</th>
              <th className="py-2.5 px-3"></th>
            </tr>
          </thead>
          <tbody>
            {clients.map((c) => (
              <tr key={c.id} className="border-t border-border hover:bg-accent-soft transition-colors">
                <td className="py-2.5 px-3">{c.name}</td>
                <td className="py-2.5 px-3 font-mono text-xs">{(c.scopes ?? []).join(" ")}</td>
                <td className="py-2.5 px-3 font-mono text-xs">{c.allowed_fields.join(" ")}</td>
                <td className="py-2.5 px-3">
                  {c.rate_limit_max}/{c.rate_limit_window_seconds}s
                </td>
                <td className="py-2.5 px-3">
                  <Button
                    variant="danger"
                    onClick={() => {
                      if (!confirm("Revoke this API client?")) return;
                      api.apiClients
                        .remove(c.id)
                        .then(() => {
                          setFlash({ msg: "API client revoked", kind: "ok" });
                          refresh();
                        })
                        .catch(() => setFlash({ msg: "Could not revoke API client", kind: "error" }));
                    }}
                  >
                    Revoke
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </TableWrap>
    </AppShell>
  );
}
