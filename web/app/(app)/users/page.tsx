"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Users as UsersIcon } from "lucide-react";
import { Badge, Button, Flash, H1, Input, Label, Pagination, Panel, TableWrap } from "@/components/ui";
import { api, ApiError, User } from "@/lib/api";
import { formatDate } from "@/lib/time";
import { useCan } from "@/lib/current-user";

const PAGE_SIZE = 50;

export default function UsersPage() {
  const canManage = useCan("users", "manage");
  const [users, setUsers] = useState<User[]>([]);
  const [offset, setOffset] = useState(0);
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [employeeId, setEmployeeId] = useState("");
  const [flash, setFlash] = useState<{ msg: string; kind: "ok" | "error" } | null>(null);

  function refresh() {
    api.users.list(PAGE_SIZE, offset).then((r) => setUsers(r.users ?? []));
  }

  useEffect(refresh, [offset]);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api.users.create(username, email, employeeId || undefined);
      setFlash({ msg: "User created with the default password; they'll be asked to change it on first login.", kind: "ok" });
      setUsername("");
      setEmail("");
      setEmployeeId("");
      setOffset(0);
      api.users.list(PAGE_SIZE, 0).then((r) => setUsers(r.users ?? []));
    } catch (err) {
      setFlash({ msg: err instanceof ApiError ? err.message : "Could not create user", kind: "error" });
    }
  }

  return (
    <>
      <H1 icon={<UsersIcon size={20} />}>Users</H1>
      {flash && <Flash message={flash.msg} kind={flash.kind} />}

      {canManage && (
        <Panel className="mb-5">
          <h2 className="text-sm font-semibold mb-1">Add user</h2>
          <form onSubmit={handleCreate}>
            <Label>Username</Label>
            <Input value={username} onChange={(e) => setUsername(e.target.value)} required />
            <Label>Email</Label>
            <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
            <Label>Employee ID (optional)</Label>
            <Input value={employeeId} onChange={(e) => setEmployeeId(e.target.value)} placeholder="e.g. EMP-00123" />
            <p className="text-xs text-muted mt-1">
              New accounts start with the organization&apos;s default password (set in Settings) and
              must change it on first login.
            </p>
            <Button type="submit" className="mt-4">
              Create
            </Button>
          </form>
        </Panel>
      )}

      <TableWrap>
        <table className="w-full border-collapse min-w-[480px]">
          <thead>
            <tr className="text-left text-[11.5px] uppercase tracking-wide text-muted">
              <th className="py-2.5 px-3">Username</th>
              <th className="py-2.5 px-3">Employee ID</th>
              <th className="py-2.5 px-3">Email</th>
              <th className="py-2.5 px-3">Status</th>
              <th className="py-2.5 px-3">MFA</th>
              <th className="py-2.5 px-3">Created</th>
              <th className="py-2.5 px-3"></th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id} className="border-t border-border hover:bg-accent-soft transition-colors">
                <td className="py-2.5 px-3">
                  <Link href={`/users/detail?id=${u.id}`} className="font-medium">
                    {u.username}
                  </Link>
                </td>
                <td className="py-2.5 px-3 text-muted font-mono text-xs">{u.employee_id}</td>
                <td className="py-2.5 px-3">{u.email}</td>
                <td className="py-2.5 px-3">
                  <Badge tone={u.status === "active" ? "ok" : "danger"}>{u.status}</Badge>
                </td>
                <td className="py-2.5 px-3">{u.mfa_enabled ? "yes" : "no"}</td>
                <td className="py-2.5 px-3 text-muted">{formatDate(u.created_at)}</td>
                <td className="py-2.5 px-3">
                  <Link href={`/users/detail?id=${u.id}`} className="text-accent text-sm">
                    Manage
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </TableWrap>
      <Pagination
        offset={offset}
        limit={PAGE_SIZE}
        count={users.length}
        onPrev={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
        onNext={() => setOffset((o) => o + PAGE_SIZE)}
      />
    </>
  );
}
