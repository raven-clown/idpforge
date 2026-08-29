"use client";

import { Fragment, useEffect, useMemo, useState } from "react";
import { Cpu } from "lucide-react";
import { Badge, Button, Flash, H1, H2, Input, Label, Pagination, Panel, SelectOrCustom, TableWrap } from "@/components/ui";
import { api, ApiError, Device, DeviceEvent } from "@/lib/api";
import { formatDate, formatDateTime } from "@/lib/time";
import { useCan } from "@/lib/current-user";

const DEVICE_TYPES = ["card_reader", "face_2d", "face_3d", "fingerprint", "door_controller", "kiosk"];
const EVENTS_PAGE_SIZE = 50;
const NO_FOLDER = "(no folder)";

function splitCSV(s: string): string[] {
  return s.split(",").map((v) => v.trim()).filter(Boolean);
}

export default function IotPage() {
  const canManage = useCan("iot", "manage");
  const [devices, setDevices] = useState<Device[]>([]);
  const [events, setEvents] = useState<DeviceEvent[]>([]);
  const [eventsOffset, setEventsOffset] = useState(0);
  const [name, setName] = useState("");
  const [deviceType, setDeviceType] = useState("");
  const [location, setLocation] = useState("");
  const [folder, setFolder] = useState("");
  const [allowedIps, setAllowedIps] = useState("");
  const [newKey, setNewKey] = useState("");
  const [flash, setFlash] = useState<{ msg: string; kind: "ok" | "error" } | null>(null);

  function refresh() {
    api.iot.devices().then((r) => setDevices(r.devices ?? []));
    api.iot.events(`?limit=${EVENTS_PAGE_SIZE}&offset=${eventsOffset}`).then((r) => setEvents(r.events ?? []));
  }
  useEffect(refresh, [eventsOffset]);

  const groupedDevices = useMemo(() => {
    const groups = new Map<string, Device[]>();
    for (const d of devices) {
      const key = d.folder || NO_FOLDER;
      groups.set(key, [...(groups.get(key) ?? []), d]);
    }
    return [...groups.entries()].sort(([a], [b]) => a.localeCompare(b));
  }, [devices]);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      const { api_key } = await api.iot.createDevice({
        name,
        device_type: deviceType,
        location,
        folder: folder.trim() || undefined,
        allowed_ips: splitCSV(allowedIps),
      });
      setNewKey(api_key);
      setFlash({ msg: "Device registered", kind: "ok" });
      setName("");
      setDeviceType("");
      setLocation("");
      setFolder("");
      setAllowedIps("");
      refresh();
    } catch (err) {
      setFlash({ msg: err instanceof ApiError ? err.message : "Could not register device", kind: "error" });
    }
  }

  return (
    <>
      <H1 icon={<Cpu size={20} />}>IoT devices</H1>
      {flash && <Flash message={flash.msg} kind={flash.kind} />}

      {newKey && (
        <Panel>
          <strong className="text-sm">New device key created. Copy it now, it cannot be shown again:</strong>
          <div className="bg-input-bg border border-dashed border-accent rounded-lg p-3.5 mt-3 font-mono text-[13px] break-all">
            {newKey}
          </div>
          <p className="text-muted text-xs mt-2">
            The reader sends this as the <code>X-Device-Key</code> header when it calls <code>POST /device/v1/checkin</code>.
          </p>
        </Panel>
      )}

      <Panel>
        <h2 className="text-sm font-semibold mb-1">How a reader connects</h2>
        <ol className="text-xs text-muted space-y-2 list-decimal list-inside">
          <li>Register the device below to get its key (shown once, like an API client&apos;s).</li>
          <li>
            Matching happens on the device itself (face/fingerprint/card), same as WebAuthn --
            this server never receives or stores a raw scan or image. The device only sends the
            <em> reference</em> it already resolved locally (a card number, or its own template ID
            for that face/fingerprint).
          </li>
          <li>
            To check someone in, the device calls <code>POST /device/v1/checkin</code> with its
            key as <code>X-Device-Key</code>:
            <pre className="bg-input-bg border border-border rounded-lg p-3 mt-1.5 overflow-x-auto">
{`{
  "credentials": [{ "credential_type": "face_2d", "credential_ref": "TEMPLATE-1234" }],
  "event_type": "door_access",
  "resource": "front-door"
}`}
            </pre>
            The response tells the device whether it&apos;s allowed and whether this already
            happened today, so the device (or a door controller/kiosk behind it) applies its own
            policy -- unlock the door, dispense a discount, log the entry, whatever it&apos;s for.
          </li>
          <li>
            To enroll a <em>new</em> reference for someone (e.g. after the device confirms a
            high-confidence match during an enrollment flow), an admin adds it on that
            user&apos;s detail page, or a scoped API client with <code>users:manage</code> calls{" "}
            <code>POST /api/v1/users/:id/device-credentials</code>. There&apos;s no self-enrollment
            path for a device to write its own credential yet -- ask if you need one; it&apos;s a
            design question worth deciding deliberately (should hardware be trusted to enroll
            without a human approving it?).
          </li>
        </ol>
      </Panel>

      {canManage && (
        <Panel>
          <h2 className="text-sm font-semibold mb-1">Register device</h2>
          <form onSubmit={handleCreate}>
            <Label>Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="front-door, canteen-kiosk-1" required />
            <Label>Type</Label>
            <SelectOrCustom value={deviceType} onChange={setDeviceType} options={DEVICE_TYPES} placeholder="Custom device type" />
            <Label>Location</Label>
            <Input value={location} onChange={(e) => setLocation(e.target.value)} />
            <Label>Folder (optional, for organizing the list below)</Label>
            <Input value={folder} onChange={(e) => setFolder(e.target.value)} placeholder="e.g. Building A doors" />
            <p className="text-muted text-xs mt-1">
              Organizational only. Each device still gets its own key regardless of folder --
              compromising one reader shouldn&apos;t compromise every door.
            </p>
            <Label>Allowed IPs / CIDRs (comma-separated, blank = unrestricted)</Label>
            <Input value={allowedIps} onChange={(e) => setAllowedIps(e.target.value)} placeholder="10.0.5.0/24" />
            <Button type="submit" className="mt-4">
              Register
            </Button>
          </form>
        </Panel>
      )}

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
            {groupedDevices.map(([folderName, group]) => (
              <Fragment key={folderName}>
                <tr className="border-t border-border bg-accent-soft/40">
                  <td colSpan={5} className="py-1.5 px-3 text-[11px] font-semibold text-muted uppercase tracking-wide">
                    {folderName} ({group.length})
                  </td>
                </tr>
                {group.map((d) => (
                  <tr key={d.id} className="border-t border-border hover:bg-accent-soft transition-colors">
                    <td className="py-2.5 px-3">{d.name}</td>
                    <td className="py-2.5 px-3">{d.device_type}</td>
                    <td className="py-2.5 px-3 text-muted">{d.location}</td>
                    <td className="py-2.5 px-3">{d.enabled ? "yes" : "no"}</td>
                    <td className="py-2.5 px-3 text-muted">{formatDate(d.created_at)}</td>
                  </tr>
                ))}
              </Fragment>
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
                  <td className="py-2 px-3 font-mono text-xs">{formatDateTime(e.timestamp)}</td>
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
        <Pagination
          offset={eventsOffset}
          limit={EVENTS_PAGE_SIZE}
          count={events.length}
          onPrev={() => setEventsOffset((o) => Math.max(0, o - EVENTS_PAGE_SIZE))}
          onNext={() => setEventsOffset((o) => o + EVENTS_PAGE_SIZE)}
        />
      </Panel>
    </>
  );
}
