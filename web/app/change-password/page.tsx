"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ShieldHalf } from "lucide-react";
import { ApiError } from "@/lib/api";
import { Button, Input, Label, Panel } from "@/components/ui";

export default function ChangePasswordPage() {
  const router = useRouter();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (newPassword !== confirm) {
      setError("New passwords do not match");
      return;
    }
    setBusy(true);
    try {
      const res = await fetch("/api/v1/change-password", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new ApiError(res.status, body.message || body.error || "Could not change password");
      }
      router.replace("/");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not change password");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-5">
      <div className="w-full max-w-[380px] animate-fade-in">
        <div className="flex items-center justify-center gap-2.5 text-xl font-bold mb-2">
          <ShieldHalf className="text-accent" size={22} />
          Password change required
        </div>
        <p className="text-muted text-sm text-center mb-5">
          Your password has expired and must be changed before continuing.
        </p>
        <Panel className="shadow-lg">
          {error && (
            <div className="px-4 py-2.5 rounded-lg text-sm mb-4 bg-danger-soft text-danger">{error}</div>
          )}
          <form onSubmit={handleSubmit}>
            <Label>Current password</Label>
            <Input type="password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} required />
            <Label>New password (at least 8 characters)</Label>
            <Input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} required minLength={8} />
            <Label>Confirm new password</Label>
            <Input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} required minLength={8} />
            <Button type="submit" disabled={busy} className="w-full justify-center mt-5">
              {busy ? "Saving..." : "Change password"}
            </Button>
          </form>
        </Panel>
      </div>
    </div>
  );
}
