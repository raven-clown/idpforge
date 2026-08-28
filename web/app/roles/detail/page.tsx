"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { ShieldCheck } from "lucide-react";
import AppShell from "@/components/AppShell";
import { Button, Flash, H1, Panel, TableWrap } from "@/components/ui";
import { api, ApiError, PermissionRecord, Role } from "@/lib/api";

function RoleDetail({ id }: { id: string }) {
  const [role, setRole] = useState<Role | null>(null);
  const [granted, setGranted] = useState<PermissionRecord[]>([]);
  const [all, setAll] = useState<PermissionRecord[]>([]);
  const [selected, setSelected] = useState("");
  const [flash, setFlash] = useState<{ msg: string; kind: "ok" | "error" } | null>(null);

  function refresh() {
    api.rbac.getRole(id).then(setRole);
    api.rbac.rolePermissions(id).then((r) => setGranted(r.permissions ?? []));
    api.rbac.permissions().then((r) => setAll(r.permissions ?? []));
  }
  useEffect(refresh, [id]);

  function report(action: () => Promise<unknown>, okMsg: string, failMsg: string) {
    action()
      .then(() => {
        setFlash({ msg: okMsg, kind: "ok" });
        refresh();
      })
      .catch((err) => setFlash({ msg: err instanceof ApiError ? err.message : failMsg, kind: "error" }));
  }

  return (
    <>
      <H1 icon={<ShieldCheck size={20} />}>{role ? role.name : "Role"}</H1>
      {flash && <Flash message={flash.msg} kind={flash.kind} />}
      <Panel>
        <TableWrap>
          <table className="w-full border-collapse">
            <thead>
              <tr className="text-left text-[11.5px] uppercase tracking-wide text-muted">
                <th className="py-2 px-3">Resource</th>
                <th className="py-2 px-3">Action</th>
                <th className="py-2 px-3"></th>
              </tr>
            </thead>
            <tbody>
              {granted.map((p) => (
                <tr key={p.id} className="border-t border-border">
                  <td className="py-2 px-3">{p.resource}</td>
                  <td className="py-2 px-3">{p.action}</td>
                  <td className="py-2 px-3">
                    <Button
                      variant="secondary"
                      onClick={() =>
                        report(
                          () => api.rbac.revokePermission(id, p.id),
                          "Permission revoked",
                          "Could not revoke permission"
                        )
                      }
                    >
                      Revoke
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableWrap>
        <div className="flex gap-2.5 items-end mt-3">
          <div className="flex-1">
            <select
              value={selected}
              onChange={(e) => setSelected(e.target.value)}
              className="w-full px-3 py-2 bg-input-bg border border-border rounded-lg text-sm text-text"
            >
              <option value="">Select a permission</option>
              {all.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.resource}:{p.action}
                </option>
              ))}
            </select>
          </div>
          <Button
            onClick={() =>
              selected &&
              report(() => api.rbac.grantPermission(id, selected), "Permission granted", "Could not grant permission")
            }
          >
            Grant
          </Button>
        </div>
      </Panel>
      <p className="mt-4">
        <Link href="/roles" className="text-accent text-sm">
          &larr; back to roles
        </Link>
      </p>
    </>
  );
}

function RoleDetailInner() {
  const params = useSearchParams();
  const id = params.get("id") ?? "";
  if (!id) return <p className="text-muted text-sm">No role selected.</p>;
  return <RoleDetail id={id} />;
}

export default function RoleDetailPage() {
  return (
    <AppShell>
      <Suspense fallback={<p className="text-muted text-sm">Loading...</p>}>
        <RoleDetailInner />
      </Suspense>
    </AppShell>
  );
}
