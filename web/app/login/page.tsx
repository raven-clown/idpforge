"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ShieldHalf } from "lucide-react";
import { api, ApiError } from "@/lib/api";
import { Button, Input, Label, Panel } from "@/components/ui";

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [mfaCode, setMfaCode] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      const result = await api.login(username, password, mfaCode);
      router.replace(result.password_change_required ? "/change-password" : "/");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Login failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-5">
      <div className="w-full max-w-[360px] animate-fade-in">
        <div className="flex items-center justify-center gap-2.5 text-xl font-bold mb-5">
          <ShieldHalf className="text-accent" size={22} />
          IdpForge
        </div>
        <Panel className="shadow-lg">
          {error && (
            <div className="px-4 py-2.5 rounded-lg text-sm mb-4 bg-danger-soft text-danger">{error}</div>
          )}
          <form onSubmit={handleSubmit}>
            <Label>Username</Label>
            <Input value={username} onChange={(e) => setUsername(e.target.value)} required autoFocus />
            <Label>Password</Label>
            <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
            <Label>MFA code (only if MFA is enabled on your account)</Label>
            <Input
              value={mfaCode}
              onChange={(e) => setMfaCode(e.target.value)}
              placeholder="6-digit code"
              inputMode="numeric"
              autoComplete="one-time-code"
            />
            <Button type="submit" disabled={busy} className="w-full justify-center mt-5">
              {busy ? "Signing in..." : "Sign in"}
            </Button>
          </form>
        </Panel>
      </div>
    </div>
  );
}
