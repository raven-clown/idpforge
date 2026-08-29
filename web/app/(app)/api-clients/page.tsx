"use client";

import { Fragment, useEffect, useMemo, useState } from "react";
import { KeyRound } from "lucide-react";
import { Button, Flash, H1, Input, Label, Panel, TableWrap } from "@/components/ui";
import { api, ApiClient, ApiError } from "@/lib/api";

const NO_FOLDER = "(no folder)";

// Every resource:action pair the built-in routes actually check (kept in
// sync with internal/bootstrap/bootstrap.go's adminPermissions list) --
// shown as checkboxes instead of asking an admin to type scope strings
// from memory.
const SCOPE_OPTIONS = [
  { resource: "users", actions: ["read", "manage"] },
  { resource: "rbac", actions: ["manage"] },
  { resource: "iot", actions: ["read", "manage"] },
  { resource: "api_clients", actions: ["manage"] },
  { resource: "audit", actions: ["read"] },
  { resource: "metrics", actions: ["read"] },
  { resource: "settings", actions: ["read"] },
  { resource: "announcements", actions: ["manage"] },
];

// The JSON fields a User object can expose through /external/v1.
const FIELD_OPTIONS = [
  "id",
  "username",
  "email",
  "status",
  "mfa_enabled",
  "source",
  "external_id",
  "avatar_url",
  "force_password_change",
  "created_at",
  "updated_at",
  "last_login_at",
];

function toggle(list: string[], value: string): string[] {
  return list.includes(value) ? list.filter((v) => v !== value) : [...list, value];
}

export default function ApiClientsPage() {
  const [clients, setClients] = useState<ApiClient[]>([]);
  const [name, setName] = useState("");
  const [folder, setFolder] = useState("");
  const [scopes, setScopes] = useState<string[]>([]);
  const [allowedFields, setAllowedFields] = useState<string[]>(["id", "username", "email"]);
  const [allowedIps, setAllowedIps] = useState("");
  const [rateLimitMax, setRateLimitMax] = useState(60);
  const [rateLimitWindow, setRateLimitWindow] = useState(60);
  const [newKey, setNewKey] = useState("");
  const [flash, setFlash] = useState<{ msg: string; kind: "ok" | "error" } | null>(null);

  function refresh() {
    api.apiClients.list().then((r) => setClients(r.clients ?? []));
  }
  useEffect(refresh, []);

  const grouped = useMemo(() => {
    const groups = new Map<string, ApiClient[]>();
    for (const c of clients) {
      const key = c.folder || NO_FOLDER;
      groups.set(key, [...(groups.get(key) ?? []), c]);
    }
    return [...groups.entries()].sort(([a], [b]) => a.localeCompare(b));
  }, [clients]);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      const { api_key } = await api.apiClients.create({
        name,
        folder: folder.trim() || undefined,
        scopes,
        allowed_fields: allowedFields,
        allowed_ips: allowedIps.split(",").map((v) => v.trim()).filter(Boolean),
        rate_limit_max: rateLimitMax,
        rate_limit_window_seconds: rateLimitWindow,
      });
      setNewKey(api_key);
      setFlash({ msg: "API client created", kind: "ok" });
      setName("");
      setFolder("");
      setScopes([]);
      setAllowedIps("");
      refresh();
    } catch (err) {
      setFlash({ msg: err instanceof ApiError ? err.message : "Could not create API client", kind: "error" });
    }
  }

  return (
    <>
      <H1 icon={<KeyRound size={20} />}>API clients</H1>
      {flash && <Flash message={flash.msg} kind={flash.kind} />}

      {newKey && (
        <Panel>
          <strong className="text-sm">New API key created. Copy it now, it cannot be shown again:</strong>
          <div className="bg-input-bg border border-dashed border-accent rounded-lg p-3.5 mt-3 font-mono text-[13px] break-all">
            {newKey}
          </div>
          <p className="text-sm font-semibold mt-4">How to use it</p>
          <p className="text-muted text-xs mt-1">
            Send it as the <code>X-API-Key</code> header. Which base path to call depends on what you
            granted above: scopes work against the full admin API, allowed fields against the simple
            read-only path.
          </p>
          <pre className="bg-input-bg border border-border rounded-lg p-3 mt-2 text-[11.5px] overflow-x-auto">
{`curl -H "X-API-Key: ${newKey}" \\
  ${typeof window !== "undefined" ? window.location.origin : ""}/api/v1/users`}
          </pre>
          <pre className="bg-input-bg border border-border rounded-lg p-3 mt-2 text-[11.5px] overflow-x-auto">
{`curl -H "X-API-Key: ${newKey}" \\
  ${typeof window !== "undefined" ? window.location.origin : ""}/external/v1/users/<user-id>`}
          </pre>
        </Panel>
      )}

      <Panel>
        <h2 className="text-sm font-semibold mb-1">New API client</h2>
        <form onSubmit={handleCreate}>
          <Label>Name</Label>
          <Input value={name} onChange={(e) => setName(e.target.value)} required placeholder="e.g. Onboarding automation" />

          <Label>Folder (optional, for organizing the list below)</Label>
          <Input value={folder} onChange={(e) => setFolder(e.target.value)} placeholder="e.g. Team Frontend" />
          <p className="text-muted text-xs mt-1">
            Purely organizational -- it groups related clients in the list, it doesn&apos;t grant
            anything by itself. If several people/services genuinely need the exact same access,
            it&apos;s fine to hand out this one client&apos;s key to all of them; if you want each
            caller individually revocable and traceable in the audit log, create one client per
            caller and give them the same folder name.
          </p>

          <Label>Scopes (full admin API access, /api/v1)</Label>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-x-4 gap-y-2 p-3 bg-input-bg border border-border rounded-lg">
            {SCOPE_OPTIONS.map((group) =>
              group.actions.map((action) => {
                const scope = `${group.resource}:${action}`;
                return (
                  <label key={scope} className="flex items-center gap-2 text-xs cursor-pointer">
                    <input
                      type="checkbox"
                      checked={scopes.includes(scope)}
                      onChange={() => setScopes((s) => toggle(s, scope))}
                      className="accent-accent"
                    />
                    <span className="font-mono">{scope}</span>
                  </label>
                );
              })
            )}
          </div>

          <Label>Allowed fields (simple read-only path, /external/v1)</Label>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-x-4 gap-y-2 p-3 bg-input-bg border border-border rounded-lg">
            {FIELD_OPTIONS.map((field) => (
              <label key={field} className="flex items-center gap-2 text-xs cursor-pointer">
                <input
                  type="checkbox"
                  checked={allowedFields.includes(field)}
                  onChange={() => setAllowedFields((f) => toggle(f, field))}
                  className="accent-accent"
                />
                <span className="font-mono">{field}</span>
              </label>
            ))}
          </div>

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
            {grouped.map(([folderName, group]) => (
              <Fragment key={folderName}>
                <tr key={`h-${folderName}`} className="border-t border-border bg-accent-soft/40">
                  <td colSpan={5} className="py-1.5 px-3 text-[11px] font-semibold text-muted uppercase tracking-wide">
                    {folderName} ({group.length})
                  </td>
                </tr>
                {group.map((c) => (
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
              </Fragment>
            ))}
          </tbody>
        </table>
      </TableWrap>
    </>
  );
}
