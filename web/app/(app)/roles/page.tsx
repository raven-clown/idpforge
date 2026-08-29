"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ShieldCheck } from "lucide-react";
import { Button, Flash, H1, H2, Input, Label, Panel, TableWrap } from "@/components/ui";
import { api, ApiError, Group, PermissionRecord, Role } from "@/lib/api";

export default function RolesPage() {
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<PermissionRecord[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [roleName, setRoleName] = useState("");
  const [roleDesc, setRoleDesc] = useState("");
  const [permResource, setPermResource] = useState("");
  const [permAction, setPermAction] = useState("");
  const [groupName, setGroupName] = useState("");
  const [groupParent, setGroupParent] = useState("");
  const [flash, setFlash] = useState<{ msg: string; kind: "ok" | "error" } | null>(null);

  function refresh() {
    api.rbac.roles().then((r) => setRoles(r.roles ?? []));
    api.rbac.permissions().then((r) => setPermissions(r.permissions ?? []));
    api.rbac.groups().then((r) => setGroups(r.groups ?? []));
  }
  useEffect(refresh, []);

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
      <H1 icon={<ShieldCheck size={20} />}>Roles &amp; permissions</H1>
      {flash && <Flash message={flash.msg} kind={flash.kind} />}

      <Panel>
        <h2 className="text-sm font-semibold mb-1">Add role</h2>
        <Label>Name</Label>
        <Input value={roleName} onChange={(e) => setRoleName(e.target.value)} />
        <Label>Description</Label>
        <Input value={roleDesc} onChange={(e) => setRoleDesc(e.target.value)} />
        <Button
          className="mt-4"
          onClick={() => {
            if (!roleName) return;
            report(() => api.rbac.createRole(roleName, roleDesc), "Role created", "Could not create role");
            setRoleName("");
            setRoleDesc("");
          }}
        >
          Create
        </Button>
      </Panel>

      <TableWrap>
        <table className="w-full border-collapse min-w-[420px]">
          <thead>
            <tr className="text-left text-[11.5px] uppercase tracking-wide text-muted">
              <th className="py-2.5 px-3">Role</th>
              <th className="py-2.5 px-3">Description</th>
              <th className="py-2.5 px-3"></th>
            </tr>
          </thead>
          <tbody>
            {roles.map((r) => (
              <tr key={r.id} className="border-t border-border hover:bg-accent-soft transition-colors">
                <td className="py-2.5 px-3">
                  <Link href={`/roles/detail?id=${r.id}`} className="font-medium">
                    {r.name}
                  </Link>
                </td>
                <td className="py-2.5 px-3 text-muted">{r.description}</td>
                <td className="py-2.5 px-3 flex gap-3 items-center">
                  <Link href={`/roles/detail?id=${r.id}`} className="text-accent text-sm">
                    Manage
                  </Link>
                  <Button
                    variant="secondary"
                    onClick={() =>
                      confirm("Delete this role?") &&
                      report(() => api.rbac.deleteRole(r.id), "Role deleted", "Could not delete role")
                    }
                  >
                    Delete
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </TableWrap>

      <H2>All permissions</H2>
      <Panel>
        <TableWrap>
          <table className="w-full border-collapse">
            <thead>
              <tr className="text-left text-[11.5px] uppercase tracking-wide text-muted">
                <th className="py-2 px-3">Resource</th>
                <th className="py-2 px-3">Action</th>
              </tr>
            </thead>
            <tbody>
              {permissions.map((p) => (
                <tr key={p.id} className="border-t border-border">
                  <td className="py-2 px-3">{p.resource}</td>
                  <td className="py-2 px-3">{p.action}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableWrap>
        <Label>Resource</Label>
        <Input value={permResource} onChange={(e) => setPermResource(e.target.value)} placeholder="users, rbac, iot, api_clients, or your own" />
        <Label>Action</Label>
        <Input value={permAction} onChange={(e) => setPermAction(e.target.value)} placeholder="read, manage" />
        <Button
          className="mt-4"
          onClick={() => {
            if (!permResource || !permAction) return;
            report(
              () => api.rbac.createPermission(permResource, permAction),
              "Permission created",
              "Could not create permission"
            );
            setPermResource("");
            setPermAction("");
          }}
        >
          Create permission
        </Button>
      </Panel>

      <H2>Groups</H2>
      <Panel>
        <TableWrap>
          <table className="w-full border-collapse">
            <thead>
              <tr className="text-left text-[11.5px] uppercase tracking-wide text-muted">
                <th className="py-2 px-3">Name</th>
                <th className="py-2 px-3">Parent group</th>
              </tr>
            </thead>
            <tbody>
              {groups.map((g) => (
                <tr key={g.id} className="border-t border-border">
                  <td className="py-2 px-3">{g.name}</td>
                  <td className="py-2 px-3 text-muted">{g.parent_group_id || "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableWrap>
        <Label>Name</Label>
        <Input value={groupName} onChange={(e) => setGroupName(e.target.value)} />
        <Label>Parent group ID (optional)</Label>
        <Input value={groupParent} onChange={(e) => setGroupParent(e.target.value)} />
        <Button
          className="mt-4"
          onClick={() => {
            if (!groupName) return;
            report(() => api.rbac.createGroup(groupName, groupParent), "Group created", "Could not create group");
            setGroupName("");
            setGroupParent("");
          }}
        >
          Create group
        </Button>
      </Panel>
    </>
  );
}
