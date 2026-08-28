"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Users as UsersIcon } from "lucide-react";
import AppShell from "@/components/AppShell";
import { Badge, Button, Flash, H1, Input, Label, Panel, TableWrap } from "@/components/ui";
import { api, ApiError, User } from "@/lib/api";

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [flash, setFlash] = useState<{ msg: string; kind: "ok" | "error" } | null>(null);

  function refresh() {
    api.users.list().then((r) => setUsers(r.users ?? []));
  }

  useEffect(refresh, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api.users.create(username, email, password);
      setFlash({ msg: "User created", kind: "ok" });
      setUsername("");
      setEmail("");
      setPassword("");
      refresh();
    } catch (err) {
      setFlash({ msg: err instanceof ApiError ? err.message : "Could not create user", kind: "error" });
    }
  }

  return (
    <AppShell>
      <H1 icon={<UsersIcon size={20} />}>Users</H1>
      {flash && <Flash message={flash.msg} kind={flash.kind} />}

      <Panel className="mb-5">
        <h2 className="text-sm font-semibold mb-1">Add user</h2>
        <form onSubmit={handleCreate}>
          <Label>Username</Label>
          <Input value={username} onChange={(e) => setUsername(e.target.value)} required />
          <Label>Email</Label>
          <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          <Label>Password</Label>
          <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
          <Button type="submit" className="mt-4">
            Create
          </Button>
        </form>
      </Panel>

      <TableWrap>
        <table className="w-full border-collapse min-w-[480px]">
          <thead>
            <tr className="text-left text-[11.5px] uppercase tracking-wide text-muted">
              <th className="py-2.5 px-3">Username</th>
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
                <td className="py-2.5 px-3">{u.email}</td>
                <td className="py-2.5 px-3">
                  <Badge tone={u.status === "active" ? "ok" : "danger"}>{u.status}</Badge>
                </td>
                <td className="py-2.5 px-3">{u.mfa_enabled ? "yes" : "no"}</td>
                <td className="py-2.5 px-3 text-muted">{new Date(u.created_at).toLocaleDateString()}</td>
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
    </AppShell>
  );
}
