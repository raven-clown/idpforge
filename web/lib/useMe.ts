"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError, User } from "./api";

// Resolves the current session on every protected page. Redirects to
// /login if unauthenticated, or to /change-password if the account's
// password has expired, before rendering anything else.
export function useMe() {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    fetch("/api/v1/me", { credentials: "include" })
      .then(async (res) => {
        if (!res.ok) throw new ApiError(res.status, "not authenticated");
        return res.json();
      })
      .then((data: { user: User; must_change_password: boolean }) => {
        if (cancelled) return;
        if (data.must_change_password && window.location.pathname !== "/change-password") {
          router.replace("/change-password");
          return;
        }
        setUser(data.user);
        setLoading(false);
      })
      .catch(() => {
        if (cancelled) return;
        router.replace("/login");
      });
    return () => {
      cancelled = true;
    };
  }, [router]);

  return { user, loading };
}

export { api };
