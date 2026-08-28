"use client";

import { ReactNode } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  LayoutDashboard,
  Users,
  ShieldCheck,
  KeyRound,
  Cpu,
  LogOut,
  Sun,
  Moon,
  ShieldHalf,
} from "lucide-react";
import { useMe, api } from "@/lib/useMe";

const NAV = [
  { href: "/", label: "Dashboard", icon: LayoutDashboard },
  { href: "/users", label: "Users", icon: Users },
  { href: "/roles", label: "Roles & permissions", icon: ShieldCheck },
  { href: "/api-clients", label: "API clients", icon: KeyRound },
  { href: "/iot", label: "IoT devices", icon: Cpu },
];

export default function AppShell({ children }: { children: ReactNode }) {
  const { user, loading } = useMe();
  const pathname = usePathname();
  const router = useRouter();

  function toggleTheme() {
    const next = !document.documentElement.classList.contains("dark");
    document.documentElement.classList.toggle("dark", next);
    try {
      localStorage.setItem("idpforge-theme", next ? "dark" : "light");
    } catch {
      // storage unavailable, theme just won't persist
    }
  }

  async function logout() {
    await api.logout().catch(() => {});
    router.replace("/login");
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center text-muted text-sm">
        Loading...
      </div>
    );
  }

  return (
    <div className="flex min-h-screen">
      <aside className="w-56 bg-panel border-r border-border p-3 flex flex-col sticky top-0 h-screen animate-fade-in max-md:w-full max-md:h-auto max-md:flex-row max-md:items-center max-md:overflow-x-auto max-md:sticky-0">
        <div className="flex items-center gap-2 px-2.5 pb-5 font-bold text-base tracking-tight max-md:pb-0 max-md:pr-4">
          <ShieldHalf size={20} className="text-accent shrink-0" />
          <span className="max-md:hidden">IdpForge</span>
        </div>
        <nav className="flex flex-col gap-0.5 flex-1 max-md:flex-row">
          {NAV.map(({ href, label, icon: Icon }) => {
            const active = pathname === href;
            return (
              <Link
                key={href}
                href={href}
                className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-[13.5px] font-medium whitespace-nowrap transition-colors ${
                  active ? "text-accent bg-accent-soft" : "text-muted hover:text-text hover:bg-accent-soft"
                }`}
              >
                <Icon size={16} className="opacity-85 shrink-0" />
                <span className="max-md:hidden">{label}</span>
              </Link>
            );
          })}
        </nav>
        <div className="mt-auto pt-2.5 border-t border-border max-md:mt-0 max-md:pt-0 max-md:border-t-0 max-md:border-l max-md:pl-2 max-md:ml-2">
          <button
            onClick={toggleTheme}
            className="flex items-center gap-2.5 w-full px-3 py-2 rounded-lg text-[13.5px] font-medium text-muted hover:text-text hover:bg-accent-soft transition-colors"
          >
            <Sun size={16} className="hidden dark:block" />
            <Moon size={16} className="dark:hidden" />
            <span className="max-md:hidden">Toggle theme</span>
          </button>
          <button
            onClick={logout}
            className="flex items-center gap-2.5 w-full px-3 py-2 rounded-lg text-[13.5px] font-medium text-accent hover:bg-accent-soft transition-colors"
          >
            <LogOut size={16} />
            <span className="max-md:hidden">Log out</span>
          </button>
        </div>
      </aside>
      <main className="flex-1 p-7 md:p-10 max-w-[1200px] animate-fade-in">
        {user && children}
      </main>
    </div>
  );
}
