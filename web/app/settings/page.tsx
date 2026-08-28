"use client";

import { useEffect, useState } from "react";
import { Settings as SettingsIcon } from "lucide-react";
import AppShell from "@/components/AppShell";
import { H1, H2, Panel, TableWrap } from "@/components/ui";
import { api, Settings } from "@/lib/api";

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <tr className="border-b border-border last:border-b-0">
      <td className="py-2 px-3 text-muted w-56">{label}</td>
      <td className="py-2 px-3 font-mono text-[13px]">{value}</td>
    </tr>
  );
}

export default function SettingsPage() {
  const [settings, setSettings] = useState<Settings | null>(null);

  useEffect(() => {
    api.settings().then(setSettings);
  }, []);

  return (
    <AppShell>
      <H1 icon={<SettingsIcon size={20} />}>Settings</H1>
      <p className="text-muted text-sm mb-5">
        Read-only. Almost everything here is set via environment variables and needs a restart to
        take effect, the same as nginx.conf or postgresql.conf.
      </p>

      {settings && (
        <>
          <Panel>
            <TableWrap>
              <table className="w-full border-collapse">
                <tbody>
                  <Row label="Environment" value={settings.env} />
                  <Row label="Listen address" value={settings.http.listen_addr} />
                  <Row label="Base URL" value={settings.http.base_url} />
                </tbody>
              </table>
            </TableWrap>
          </Panel>

          <H2>Database</H2>
          <Panel>
            <TableWrap>
              <table className="w-full border-collapse">
                <tbody>
                  <Row label="Driver" value={settings.database.driver} />
                  <Row label="DSN" value={settings.database.dsn} />
                </tbody>
              </table>
            </TableWrap>
          </Panel>

          <H2>Redis / cache</H2>
          <Panel>
            <TableWrap>
              <table className="w-full border-collapse">
                <tbody>
                  <Row label="Enabled" value={String(settings.redis.enabled)} />
                  <Row label="Address" value={settings.redis.addr} />
                </tbody>
              </table>
            </TableWrap>
          </Panel>

          <H2>Rate limiting</H2>
          <Panel>
            <TableWrap>
              <table className="w-full border-collapse">
                <tbody>
                  <Row label="Enabled" value={String(settings.rate_limit.enabled)} />
                  <Row label="Global limit" value={`${settings.rate_limit.max} / ${settings.rate_limit.window_seconds}s`} />
                  <Row label="Login limit" value={`${settings.rate_limit.login_max} / ${settings.rate_limit.login_window_seconds}s`} />
                </tbody>
              </table>
            </TableWrap>
          </Panel>

          <H2>Captcha</H2>
          <Panel>
            <TableWrap>
              <table className="w-full border-collapse">
                <tbody>
                  <Row label="Provider" value={settings.captcha.provider} />
                </tbody>
              </table>
            </TableWrap>
          </Panel>

          <H2>OIDC</H2>
          <Panel>
            <TableWrap>
              <table className="w-full border-collapse">
                <tbody>
                  <Row label="Issuer" value={settings.oidc.issuer} />
                  <Row label="Access token TTL" value={`${settings.oidc.access_token_ttl_minutes} min`} />
                  <Row label="ID token TTL" value={`${settings.oidc.id_token_ttl_minutes} min`} />
                  <Row label="Refresh token TTL" value={`${settings.oidc.refresh_token_ttl_hours} hr`} />
                </tbody>
              </table>
            </TableWrap>
          </Panel>

          <H2>Backup</H2>
          <Panel>
            <TableWrap>
              <table className="w-full border-collapse">
                <tbody>
                  <Row label="Enabled" value={String(settings.backup.enabled)} />
                  <Row label="Directory" value={settings.backup.dir} />
                  <Row label="Schedule" value={settings.backup.schedule} />
                  <Row label="Retention" value={`${settings.backup.retention_days} days`} />
                </tbody>
              </table>
            </TableWrap>
          </Panel>

          <H2>Storage &amp; security</H2>
          <Panel>
            <TableWrap>
              <table className="w-full border-collapse">
                <tbody>
                  <Row label="Avatar storage backend" value={settings.storage.backend} />
                  <Row
                    label="Password expiry"
                    value={settings.password_expiry_days > 0 ? `${settings.password_expiry_days} days` : "disabled"}
                  />
                </tbody>
              </table>
            </TableWrap>
          </Panel>
        </>
      )}
    </AppShell>
  );
}
