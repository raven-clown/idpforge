"use client";

import { useEffect, useState } from "react";
import { Cpu } from "lucide-react";
import AppShell from "@/components/AppShell";
import { Badge, Button, Flash, H1, H2, Input, Label, Panel, TableWrap } from "@/components/ui";
import { api, ApiError, Device, DeviceEvent } from "@/lib/api";

function splitCSV(s: string): string[] {
  return s.split(",").map((v) => v.trim()).filter(Boolean);
}

export default function IotPage() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [events, setEvents] = useState<DeviceEvent[]>([]);
  const [name, setName] = useState("");
  const [deviceType, setDeviceType] = useState("");
  const [location, setLocation] = useState("");
  const [allowedIps, setAllowedIps] = useState("");
  const [newKey, setNewKey] = useState("");
  const [flash, setFlash] = useState<{ msg: string; kind: "ok" | "error" } | null>(null);

  function refresh() {
    api.iot.devices().then((r) => setDevices(r.devices ?? []));
    api.iot.events("?limit=50").then((r) => setEvents(r.events ?? []));
  }
  useEffect(refresh, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      const { api_key } = await api.iot.createDevice({
        name,
        device_type: deviceType,
        location,
        allowed_ips: splitCSV(allowedIps),
      });
      setNewKey(api_key);
      setFlash({ msg: "Device registered", kind: "ok" });
      setName("");
      setDeviceType("");
      setLocation("");
      setAllowedIps("");
      refresh();
    } catch (err) {
      setFlash({ msg: err instanceof ApiError ? err.message : "Could not register device", kind: "error" });
    }
  }

  return (
    <AppShell>
      <H1 icon={<Cpu size={20} />}>IoT devices</H1>
      {flash && <Flash message={flash.msg} kind={flash.kind} />}

      {newKey && (
        <Panel>
          <strong className="text-sm">New device key created. Copy it now, it cannot be shown again:</strong>
          <div className="bg-input-bg border border-dashed border-accent rounded-lg p-3.5 mt-3 font-mono text-[13px] break-all">
            {newKey}
          </div>
          <p className="text-muted text-xs mt-2">
            The reader sends this as the <code>X-Device-Key</code> header when it calls <code>POST /iot/checkin</code>.
          </p>
        </Panel>
      )}

      <Panel>
        <h2 className="text-sm font-semibold mb-1">Register device</h2>
        <form onSubmit={handleCreate}>
          <Label>Name</Label>
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="front-door, canteen-kiosk-1" required />
          <Label>Type</Label>
          <Input value={deviceType} onChange={(e) => setDeviceType(e.target.value)} placeholder="card_reader, face_2d, face_3d, fingerprint" required />
          <Label>Location</Label>
          <Input value={location} onChange={(e) => setLocation(e.target.value)} />
          <Label>Allowed IPs / CIDRs (comma-separated, blank = unrestricted)</Label>
          <Input value={allowedIps} onChange={(e) => setAllowedIps(e.target.value)} placeholder="10.0.5.0/24" />
          <Button type="submit" className="mt-4">
            Register
          </Button>
        </form>
      </Panel>

      <TableWrap>
        <table className="w-full border-collapse min-w-[480px]">
          <thead>
            <tr className="text-left text-[11.5px] uppercase tracking-wide text-muted">
              <th className="py-2.5 px-3">Name</th>
              <th className="py-2.5 px-3">Type</th>
              <th className="py-2.5 px-3">Location</th>
              <th className="py-2.5 px-3">Enabled</th>
              <th className="py-2.5 px-3">Registered</th>
            </tr>
          </thead>
          <tbody>
            {devices.map((d) => (
              <tr key={d.id} className="border-t border-border hover:bg-accent-soft transition-colors">
                <td className="py-2.5 px-3">{d.name}</td>
                <td className="py-2.5 px-3">{d.device_type}</td>
                <td className="py-2.5 px-3 text-muted">{d.location}</td>
                <td className="py-2.5 px-3">{d.enabled ? "yes" : "no"}</td>
                <td className="py-2.5 px-3 text-muted">{new Date(d.created_at).toLocaleDateString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </TableWrap>

      <H2>Recent check-in events</H2>
      <Panel>
        <TableWrap>
          <table className="w-full border-collapse">
            <thead>
              <tr className="text-left text-[11.5px] uppercase tracking-wide text-muted">
                <th className="py-2 px-3">Time</th>
                <th className="py-2 px-3">Event</th>
                <th className="py-2 px-3">Resource</th>
                <th className="py-2 px-3">User</th>
                <th className="py-2 px-3">Status</th>
              </tr>
            </thead>
            <tbody>
              {events.map((e) => (
                <tr key={e.id} className="border-t border-border">
                  <td className="py-2 px-3 font-mono text-xs">
                    {new Date(e.timestamp).toISOString().slice(0, 19).replace("T", " ")} UTC
                  </td>
                  <td className="py-2 px-3">{e.event_type}</td>
                  <td className="py-2 px-3">{e.resource}</td>
                  <td className="py-2 px-3 font-mono text-xs">{e.user_id}</td>
                  <td className="py-2 px-3">
                    <Badge tone={e.status === "matched" ? "ok" : "danger"}>{e.status}</Badge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableWrap>
      </Panel>
    </AppShell>
  );
}
