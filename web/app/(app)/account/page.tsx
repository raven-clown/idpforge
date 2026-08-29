"use client";

import { useState } from "react";
import { UserCircle } from "lucide-react";
import { Button, Flash, H1, H2, Input, Label, Panel } from "@/components/ui";
import { api, ApiError } from "@/lib/api";
import { useCurrentUser } from "@/lib/current-user";
import { registerSecurityKey, webauthnSupported } from "@/lib/webauthn";

export default function AccountPage() {
  const user = useCurrentUser();
  const [flash, setFlash] = useState<{ msg: string; kind: "ok" | "error" } | null>(null);

  const [avatarFile, setAvatarFile] = useState<File | null>(null);
  const [avatarBusy, setAvatarBusy] = useState(false);

  const [enrolling, setEnrolling] = useState<{ secret: string; otpauth_url: string } | null>(null);
  const [mfaCode, setMfaCode] = useState("");
  const [disablePassword, setDisablePassword] = useState("");

  const [keyBusy, setKeyBusy] = useState(false);

  if (!user) return null;

  async function uploadAvatar() {
    if (!avatarFile) return;
    setAvatarBusy(true);
    try {
      await api.avatar.upload(user!.id, avatarFile, avatarFile.type || "application/octet-stream");
      setFlash({ msg: "Avatar updated", kind: "ok" });
      setAvatarFile(null);
      setTimeout(() => window.location.reload(), 600);
    } catch (err) {
      setFlash({ msg: err instanceof ApiError ? err.message : "Could not upload avatar", kind: "error" });
    } finally {
      setAvatarBusy(false);
    }
  }

  async function startMfaEnroll() {
    try {
      const res = await api.mfa.enroll();
      setEnrolling(res);
    } catch (err) {
      setFlash({ msg: err instanceof ApiError ? err.message : "Could not start MFA enrollment", kind: "error" });
    }
  }

  async function confirmMfa(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api.mfa.confirm(mfaCode);
      setFlash({ msg: "MFA enabled", kind: "ok" });
      setEnrolling(null);
      setMfaCode("");
      setTimeout(() => window.location.reload(), 600);
    } catch (err) {
      setFlash({ msg: err instanceof ApiError ? err.message : "Invalid code", kind: "error" });
    }
  }

  async function disableMfa(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api.mfa.disable(disablePassword);
      setFlash({ msg: "MFA disabled", kind: "ok" });
      setDisablePassword("");
      setTimeout(() => window.location.reload(), 600);
    } catch (err) {
      setFlash({ msg: err instanceof ApiError ? err.message : "Could not disable MFA", kind: "error" });
    }
  }

  async function registerKey() {
    setKeyBusy(true);
    try {
      const options = await api.webauthn.registerBegin();
      const credential = await registerSecurityKey(options);
      await api.webauthn.registerFinish(credential);
      setFlash({ msg: "Security key registered", kind: "ok" });
    } catch (err) {
      setFlash({ msg: err instanceof ApiError ? err.message : "Could not register security key", kind: "error" });
    } finally {
      setKeyBusy(false);
    }
  }

  return (
    <>
      <H1 icon={<UserCircle size={20} />}>My account</H1>
      {flash && <Flash message={flash.msg} kind={flash.kind} />}

      <Panel>
        <h2 className="text-sm font-semibold mb-1">Profile picture</h2>
        <div className="flex items-center gap-4">
          {user.avatar_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={user.avatar_url} alt="" className="w-16 h-16 rounded-full object-cover border-2 border-border" />
          ) : (
            <div className="w-16 h-16 rounded-full bg-accent-soft flex items-center justify-center text-accent font-bold text-xl">
              {user.username[0]?.toUpperCase()}
            </div>
          )}
          <div className="flex-1">
            <input
              type="file"
              accept="image/*"
              onChange={(e) => setAvatarFile(e.target.files?.[0] ?? null)}
              className="text-xs text-muted"
            />
            <Button className="mt-2" onClick={uploadAvatar} disabled={!avatarFile || avatarBusy}>
              {avatarBusy ? "Uploading..." : "Upload"}
            </Button>
          </div>
        </div>
      </Panel>

      <H2>Two-factor authentication (TOTP)</H2>
      <Panel>
        {user.mfa_enabled ? (
          <>
            <p className="text-sm text-ok mb-3">MFA is enabled on your account.</p>
            <form onSubmit={disableMfa} className="max-w-xs">
              <Label>Current password (to confirm disabling MFA)</Label>
              <Input
                type="password"
                value={disablePassword}
                onChange={(e) => setDisablePassword(e.target.value)}
                required
              />
              <Button type="submit" variant="danger" className="mt-3">
                Disable MFA
              </Button>
            </form>
          </>
        ) : enrolling ? (
          <form onSubmit={confirmMfa} className="max-w-sm">
            <p className="text-sm text-muted mb-2">
              Add this to your authenticator app (Google Authenticator, Authy, 1Password, ...). There&apos;s no QR
              scan here -- enter the secret manually, or open this link on the same device as your authenticator:
            </p>
            <div className="bg-input-bg border border-border rounded-lg p-3 font-mono text-xs break-all mb-3">
              {enrolling.secret}
            </div>
            <a href={enrolling.otpauth_url} className="text-accent text-xs break-all">
              {enrolling.otpauth_url}
            </a>
            <Label>Code from your authenticator app</Label>
            <Input value={mfaCode} onChange={(e) => setMfaCode(e.target.value)} placeholder="123456" required />
            <Button type="submit" className="mt-3">
              Confirm and enable
            </Button>
          </form>
        ) : (
          <Button onClick={startMfaEnroll}>Enable MFA</Button>
        )}
      </Panel>

      <H2>Security keys (WebAuthn)</H2>
      <Panel>
        {webauthnSupported() ? (
          <>
            <p className="text-sm text-muted mb-3">
              Register a hardware security key or platform authenticator (Touch ID, Windows Hello, ...).
            </p>
            <Button onClick={registerKey} disabled={keyBusy}>
              {keyBusy ? "Waiting for your key..." : "Register a security key"}
            </Button>
          </>
        ) : (
          <p className="text-sm text-muted">Your browser doesn&apos;t support WebAuthn.</p>
        )}
      </Panel>
    </>
  );
}
