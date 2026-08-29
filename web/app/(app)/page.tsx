"use client";

import { useEffect, useState } from "react";
import { LayoutDashboard, Users, ShieldCheck, KeyRound, Cpu } from "lucide-react";
import { H1, Panel } from "@/components/ui";
import { api } from "@/lib/api";
import { useCurrentUser } from "@/lib/current-user";

export default function DashboardPage() {
  const user = useCurrentUser();
  const [counts, setCounts] = useState({ users: 0, roles: 0, apiClients: 0, devices: 0 });

  useEffect(() => {
    Promise.all([api.users.list(), api.rbac.roles(), api.apiClients.list(), api.iot.devices()])
      .then(([u, r, c, d]) =>
        setCounts({
          users: u.users?.length ?? 0,
          roles: r.roles?.length ?? 0,
          apiClients: c.clients?.length ?? 0,
          devices: d.devices?.length ?? 0,
        })
      )
      .catch(() => {});
  }, []);

  const cards = [
    { label: "Users", value: counts.users, icon: Users },
    { label: "Roles", value: counts.roles, icon: ShieldCheck },
    { label: "API clients", value: counts.apiClients, icon: KeyRound },
    { label: "IoT devices", value: counts.devices, icon: Cpu },
  ];

  return (
    <>
      <H1 icon={<LayoutDashboard size={20} />}>Dashboard</H1>
      <div className="grid grid-cols-[repeat(auto-fit,minmax(180px,1fr))] gap-4">
        {cards.map(({ label, value, icon: Icon }) => (
          <Panel key={label}>
            <Icon className="text-accent mb-2.5" size={22} />
            <div className="text-2xl font-bold tracking-tight">{value}</div>
            <div className="text-muted text-xs mt-0.5">{label}</div>
          </Panel>
        ))}
      </div>
      {user && <p className="text-muted mt-6 text-sm">Signed in as {user.username}</p>}
    </>
  );
}
