"use client";

import { useEffect, useMemo, useState } from "react";
import { BarChart3 } from "lucide-react";
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
} from "recharts";
import { H1, Panel } from "@/components/ui";
import { api, MetricsSnapshot } from "@/lib/api";

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let value = bytes / 1024;
  let i = 0;
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024;
    i++;
  }
  return `${value.toFixed(1)} ${units[i]}`;
}

interface StoragePoint {
  time: string;
  bytes: number;
}

// Storage is a point-in-time gauge (bytes on disk right now), not a
// cumulative counter, so it's plotted directly rather than diffed like the
// request/login counters above.
function toStoragePoints(snapshots: MetricsSnapshot[]): StoragePoint[] {
  return snapshots.map((s) => ({
    time: new Date(s.timestamp).toLocaleString(undefined, {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    }),
    bytes: s.storage_bytes,
  }));
}

interface DeltaPoint {
  time: string;
  requests: number;
  loginSuccess: number;
  loginFailure: number;
  rateLimitRejections: number;
}

// Snapshots store cumulative totals since the process started; this turns
// consecutive pairs into per-interval deltas, which is what "usage over
// time" actually means (a rising cumulative line just shows uptime).
function toDeltas(snapshots: MetricsSnapshot[]): DeltaPoint[] {
  const points: DeltaPoint[] = [];
  for (let i = 1; i < snapshots.length; i++) {
    const prev = snapshots[i - 1];
    const cur = snapshots[i];
    points.push({
      time: new Date(cur.timestamp).toLocaleString(undefined, {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      }),
      requests: Math.max(0, cur.http_requests - prev.http_requests),
      loginSuccess: Math.max(0, cur.login_success - prev.login_success),
      loginFailure: Math.max(0, cur.login_failure - prev.login_failure),
      rateLimitRejections: Math.max(0, cur.rate_limit_rejections - prev.rate_limit_rejections),
    });
  }
  return points;
}

export default function UsagePage() {
  const [snapshots, setSnapshots] = useState<MetricsSnapshot[]>([]);
  const [days, setDays] = useState(30);

  useEffect(() => {
    api.metricsHistory(days).then((r) => setSnapshots(r.snapshots ?? []));
  }, [days]);

  const points = useMemo(() => toDeltas(snapshots), [snapshots]);
  const storagePoints = useMemo(() => toStoragePoints(snapshots), [snapshots]);
  const latest = snapshots[snapshots.length - 1];

  return (
    <>
      <H1 icon={<BarChart3 size={20} />}>Usage</H1>

      <div className="flex gap-2 mb-5">
        {[7, 30, 90].map((d) => (
          <button
            key={d}
            onClick={() => setDays(d)}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold border transition-colors ${
              days === d
                ? "bg-accent text-white border-accent"
                : "border-border text-muted hover:text-text hover:border-accent"
            }`}
          >
            {d} days
          </button>
        ))}
      </div>

      {latest && (
        <div className="grid grid-cols-[repeat(auto-fit,minmax(160px,1fr))] gap-4 mb-6">
          <Panel>
            <div className="text-2xl font-bold">{latest.http_requests.toLocaleString()}</div>
            <div className="text-muted text-xs mt-0.5">Total requests (lifetime)</div>
          </Panel>
          <Panel>
            <div className="text-2xl font-bold">{latest.login_success.toLocaleString()}</div>
            <div className="text-muted text-xs mt-0.5">Successful logins (lifetime)</div>
          </Panel>
          <Panel>
            <div className="text-2xl font-bold">{latest.login_failure.toLocaleString()}</div>
            <div className="text-muted text-xs mt-0.5">Failed logins (lifetime)</div>
          </Panel>
          <Panel>
            <div className="text-2xl font-bold">{latest.rate_limit_rejections.toLocaleString()}</div>
            <div className="text-muted text-xs mt-0.5">Rate limit rejections (lifetime)</div>
          </Panel>
          <Panel>
            <div className="text-2xl font-bold">{formatBytes(latest.storage_bytes)}</div>
            <div className="text-muted text-xs mt-0.5">Storage used (avatars)</div>
          </Panel>
        </div>
      )}

      <Panel>
        <h2 className="text-sm font-semibold mb-3">HTTP requests per interval</h2>
        {points.length === 0 ? (
          <p className="text-muted text-sm">
            Not enough history yet, snapshots are taken every 10 minutes. Check back shortly.
          </p>
        ) : (
          <div style={{ width: "100%", height: 260 }}>
            <ResponsiveContainer>
              <AreaChart data={points}>
                <defs>
                  <linearGradient id="reqFill" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#4f6df5" stopOpacity={0.35} />
                    <stop offset="100%" stopColor="#4f6df5" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                <XAxis dataKey="time" tick={{ fontSize: 11, fill: "var(--muted)" }} minTickGap={30} />
                <YAxis tick={{ fontSize: 11, fill: "var(--muted)" }} allowDecimals={false} />
                <Tooltip contentStyle={{ background: "var(--panel)", border: "1px solid var(--border)", borderRadius: 8, fontSize: 12 }} />
                <Area type="monotone" dataKey="requests" stroke="#4f6df5" fill="url(#reqFill)" strokeWidth={2} name="Requests" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}
      </Panel>

      <Panel>
        <h2 className="text-sm font-semibold mb-3">Logins per interval</h2>
        {points.length === 0 ? (
          <p className="text-muted text-sm">No history yet.</p>
        ) : (
          <div style={{ width: "100%", height: 260 }}>
            <ResponsiveContainer>
              <AreaChart data={points}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                <XAxis dataKey="time" tick={{ fontSize: 11, fill: "var(--muted)" }} minTickGap={30} />
                <YAxis tick={{ fontSize: 11, fill: "var(--muted)" }} allowDecimals={false} />
                <Tooltip contentStyle={{ background: "var(--panel)", border: "1px solid var(--border)", borderRadius: 8, fontSize: 12 }} />
                <Legend wrapperStyle={{ fontSize: 12 }} />
                <Area type="monotone" dataKey="loginSuccess" stroke="#1a9e6b" fill="#1a9e6b" fillOpacity={0.15} strokeWidth={2} name="Success" />
                <Area type="monotone" dataKey="loginFailure" stroke="#e5484d" fill="#e5484d" fillOpacity={0.15} strokeWidth={2} name="Failure" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}
      </Panel>

      <Panel>
        <h2 className="text-sm font-semibold mb-3">Rate limit rejections per interval</h2>
        {points.length === 0 ? (
          <p className="text-muted text-sm">No history yet.</p>
        ) : (
          <div style={{ width: "100%", height: 220 }}>
            <ResponsiveContainer>
              <AreaChart data={points}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                <XAxis dataKey="time" tick={{ fontSize: 11, fill: "var(--muted)" }} minTickGap={30} />
                <YAxis tick={{ fontSize: 11, fill: "var(--muted)" }} allowDecimals={false} />
                <Tooltip contentStyle={{ background: "var(--panel)", border: "1px solid var(--border)", borderRadius: 8, fontSize: 12 }} />
                <Area type="monotone" dataKey="rateLimitRejections" stroke="#e5484d" fill="#e5484d" fillOpacity={0.2} strokeWidth={2} name="Rejections" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}
      </Panel>

      <Panel>
        <h2 className="text-sm font-semibold mb-3">Storage used over time</h2>
        {storagePoints.length === 0 ? (
          <p className="text-muted text-sm">
            Not enough history yet, snapshots are taken every 10 minutes. Check back shortly.
          </p>
        ) : (
          <div style={{ width: "100%", height: 220 }}>
            <ResponsiveContainer>
              <AreaChart data={storagePoints}>
                <defs>
                  <linearGradient id="storageFill" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#1a9e6b" stopOpacity={0.3} />
                    <stop offset="100%" stopColor="#1a9e6b" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                <XAxis dataKey="time" tick={{ fontSize: 11, fill: "var(--muted)" }} minTickGap={30} />
                <YAxis tick={{ fontSize: 11, fill: "var(--muted)" }} tickFormatter={formatBytes} width={70} />
                <Tooltip
                  contentStyle={{ background: "var(--panel)", border: "1px solid var(--border)", borderRadius: 8, fontSize: 12 }}
                  formatter={(v) => formatBytes(Number(v))}
                />
                <Area type="monotone" dataKey="bytes" stroke="#1a9e6b" fill="url(#storageFill)" strokeWidth={2} name="Storage" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}
      </Panel>
    </>
  );
}
