"use client";

import { ReactNode, useEffect, useMemo, useRef, useState } from "react";
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
  ScrollText,
  BarChart3,
  Settings as SettingsIcon,
  Search,
  Bell,
  Megaphone,
  UserCircle,
} from "lucide-react";
import { useMe, api } from "@/lib/useMe";
import { useRealtime } from "@/lib/ws";
import { Announcement, ApiError, User } from "@/lib/api";
import { formatRelative } from "@/lib/time";
import { CurrentUserProvider } from "@/lib/current-user";
import { hasPermission } from "@/lib/permissions";

// requires mirrors the permission each page's own API calls are actually
// gated on server-side (see internal/httpserver/router.go) -- undefined
// means everyone signed in can see it (Dashboard's own tiles degrade
// individually if a count they ask for 403s).
const NAV: { href: string; label: string; icon: typeof LayoutDashboard; requires?: [string, string] }[] = [
  { href: "/", label: "Dashboard", icon: LayoutDashboard },
  { href: "/users", label: "Users", icon: Users, requires: ["users", "read"] },
  { href: "/roles", label: "Roles", icon: ShieldCheck, requires: ["rbac", "manage"] },
  { href: "/api-clients", label: "API clients", icon: KeyRound, requires: ["api_clients", "manage"] },
  { href: "/iot", label: "IoT devices", icon: Cpu, requires: ["iot", "read"] },
  { href: "/usage", label: "Usage", icon: BarChart3, requires: ["metrics", "read"] },
  { href: "/audit", label: "Audit log", icon: ScrollText, requires: ["audit", "read"] },
  { href: "/settings", label: "Settings", icon: SettingsIcon, requires: ["settings", "read"] },
];

const SEEN_KEY = "idpforge-announcements-seen-at";

function useOutsideClick(onOutside: () => void) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) onOutside();
    }
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [onOutside]);
  return ref;
}

function levelDotClass(level: Announcement["level"]) {
  if (level === "critical") return "bg-danger";
  if (level === "warning") return "bg-accent";
  return "bg-ok";
}

export default function AppShell({ children }: { children: ReactNode }) {
  const { user, version, permissions, loading } = useMe();
  const pathname = usePathname();
  const router = useRouter();

  const visibleNav = useMemo(
    () => NAV.filter((item) => !item.requires || hasPermission(permissions, item.requires[0], item.requires[1])),
    [permissions]
  );

  const [searchOpen, setSearchOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const searchRef = useOutsideClick(() => setSearchOpen(false));

  const [bellOpen, setBellOpen] = useState(false);
  const [announcementList, setAnnouncementList] = useState<Announcement[]>([]);
  const [lastSeen, setLastSeen] = useState(() => {
    try {
      return Number(localStorage.getItem(SEEN_KEY) ?? 0);
    } catch {
      return 0;
    }
  });
  const bellRef = useOutsideClick(() => setBellOpen(false));

  const [profileOpen, setProfileOpen] = useState(false);
  const profileRef = useOutsideClick(() => setProfileOpen(false));

  const [composerOpen, setComposerOpen] = useState(false);
  const [composerText, setComposerText] = useState("");
  const [composerError, setComposerError] = useState("");

  useEffect(() => {
    if (!user) return;
    api.announcements.list().then((r) => setAnnouncementList(r.announcements ?? []));
  }, [user]);

  useRealtime((e) => {
    if (e.type === "announcement") {
      setAnnouncementList((prev) => [e.announcement, ...prev].slice(0, 20));
    }
  });

  const unreadCount = useMemo(
    () => announcementList.filter((a) => new Date(a.created_at).getTime() > lastSeen).length,
    [announcementList, lastSeen]
  );

  function openBell() {
    setBellOpen((v) => !v);
    if (!bellOpen) {
      const now = Date.now();
      setLastSeen(now);
      try {
        localStorage.setItem(SEEN_KEY, String(now));
      } catch {
        // storage unavailable
      }
    }
  }

  function openSearch() {
    setSearchOpen(true);
    if (allUsers.length === 0) {
      // The topbar search needs a broad set to filter client-side; the
      // paginated Users page itself asks for one page at a time instead.
      api.users.list(500, 0).then((r) => setAllUsers(r.users ?? []));
    }
  }

  const matches = useMemo(() => {
    if (!query.trim()) return [];
    const q = query.toLowerCase();
    return allUsers
      .filter((u) => u.username.toLowerCase().includes(q) || u.email.toLowerCase().includes(q))
      .slice(0, 8);
  }, [query, allUsers]);

  async function submitAnnouncement(e: React.FormEvent) {
    e.preventDefault();
    setComposerError("");
    try {
      await api.announcements.create(composerText);
      setComposerText("");
      setComposerOpen(false);
    } catch (err) {
      setComposerError(err instanceof ApiError ? err.message : "Could not post announcement");
    }
  }

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
    <div className="min-h-screen flex flex-col">
      <header className="sticky top-0 z-20 h-14 bg-panel border-b border-border flex items-center px-3 md:px-4 gap-2">
        <Link href="/" className="flex items-center gap-2 font-bold text-base tracking-tight shrink-0 pr-2">
          <ShieldHalf size={22} className="text-accent" />
          <span className="hidden sm:inline">IdpForge</span>
        </Link>

        <nav className="flex-1 flex items-center justify-center gap-0.5 overflow-x-auto">
          {visibleNav.map(({ href, label, icon: Icon }) => {
            const active = pathname === href;
            return (
              <Link
                key={href}
                href={href}
                title={label}
                className={`relative flex flex-col items-center justify-center gap-0.5 w-14 md:w-16 h-14 shrink-0 transition-colors ${
                  active ? "text-accent" : "text-muted hover:text-text hover:bg-accent-soft"
                }`}
              >
                <Icon size={19} />
                <span className="hidden md:block text-[10px] font-medium leading-none">{label}</span>
                {active && <span className="absolute bottom-0 left-2 right-2 h-[3px] rounded-t-full bg-accent" />}
              </Link>
            );
          })}
        </nav>

        <div className="flex items-center gap-1 shrink-0 relative">
          <div ref={searchRef} className="relative">
            {searchOpen ? (
              <div className="flex items-center bg-input-bg border border-border rounded-full px-3 py-1.5 w-40 sm:w-56 animate-fade-in">
                <Search size={14} className="text-muted shrink-0" />
                <input
                  autoFocus
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="Search users..."
                  className="bg-transparent outline-none text-sm px-2 w-full"
                />
              </div>
            ) : (
              <button
                onClick={openSearch}
                title="Search users"
                className="w-9 h-9 flex items-center justify-center rounded-full text-muted hover:text-text hover:bg-accent-soft transition-colors"
              >
                <Search size={17} />
              </button>
            )}
            {searchOpen && matches.length > 0 && (
              <div className="absolute right-0 mt-2 w-72 bg-panel border border-border rounded-xl shadow-lg overflow-hidden animate-fade-in">
                {matches.map((u) => (
                  <Link
                    key={u.id}
                    href={`/users/detail?id=${u.id}`}
                    onClick={() => setSearchOpen(false)}
                    className="flex flex-col px-3.5 py-2.5 hover:bg-accent-soft transition-colors border-b border-border last:border-b-0"
                  >
                    <span className="text-sm font-medium">{u.username}</span>
                    <span className="text-xs text-muted">{u.email}</span>
                  </Link>
                ))}
              </div>
            )}
          </div>

          <div ref={bellRef} className="relative">
            <button
              onClick={openBell}
              title="Notifications"
              className="relative w-9 h-9 flex items-center justify-center rounded-full text-muted hover:text-text hover:bg-accent-soft transition-colors"
            >
              <Bell size={17} />
              {unreadCount > 0 && (
                <span className="absolute top-1 right-1 min-w-[15px] h-[15px] px-[3px] rounded-full bg-danger text-white text-[9.5px] font-bold flex items-center justify-center">
                  {unreadCount > 9 ? "9+" : unreadCount}
                </span>
              )}
            </button>
            {bellOpen && (
              <div className="absolute right-0 mt-2 w-80 bg-panel border border-border rounded-xl shadow-lg overflow-hidden animate-fade-in">
                <div className="px-3.5 py-2.5 border-b border-border text-sm font-semibold flex items-center justify-between">
                  Notifications
                  {hasPermission(permissions, "announcements", "manage") && (
                    <button
                      onClick={() => setComposerOpen((v) => !v)}
                      title="Post an announcement"
                      className="text-muted hover:text-accent transition-colors"
                    >
                      <Megaphone size={15} />
                    </button>
                  )}
                </div>
                {composerOpen && (
                  <form onSubmit={submitAnnouncement} className="p-3 border-b border-border bg-accent-soft/30">
                    <textarea
                      autoFocus
                      value={composerText}
                      onChange={(e) => setComposerText(e.target.value)}
                      placeholder="Message everyone signed in..."
                      rows={2}
                      className="w-full px-2.5 py-2 bg-input-bg border border-border rounded-lg text-xs resize-none focus:outline-none focus:border-accent"
                    />
                    {composerError && <p className="text-danger text-[11px] mt-1">{composerError}</p>}
                    <button
                      type="submit"
                      disabled={!composerText.trim()}
                      className="mt-2 px-3 py-1.5 rounded-lg bg-accent text-white text-xs font-semibold disabled:opacity-40"
                    >
                      Post
                    </button>
                  </form>
                )}
                <div className="max-h-80 overflow-y-auto">
                  {announcementList.length === 0 ? (
                    <p className="text-muted text-xs px-3.5 py-6 text-center">No announcements yet</p>
                  ) : (
                    announcementList.map((a) => (
                      <div key={a.id} className="flex gap-2.5 px-3.5 py-3 border-b border-border last:border-b-0">
                        <span className={`mt-1 w-2 h-2 rounded-full shrink-0 ${levelDotClass(a.level)}`} />
                        <div className="min-w-0">
                          <p className="text-xs leading-snug break-words">{a.message}</p>
                          <p className="text-[10.5px] text-muted mt-1">{formatRelative(a.created_at)}</p>
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </div>
            )}
          </div>

          <button
            onClick={toggleTheme}
            title="Toggle theme"
            className="w-9 h-9 flex items-center justify-center rounded-full text-muted hover:text-text hover:bg-accent-soft transition-colors"
          >
            <Sun size={17} className="hidden dark:block" />
            <Moon size={17} className="dark:hidden" />
          </button>

          <div ref={profileRef} className="relative">
            <button
              onClick={() => setProfileOpen((v) => !v)}
              className="w-9 h-9 rounded-full bg-accent text-white flex items-center justify-center text-sm font-bold overflow-hidden ml-0.5"
            >
              {user?.avatar_url ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={user.avatar_url} alt="" className="w-full h-full object-cover" />
              ) : (
                user?.username?.[0]?.toUpperCase() ?? "?"
              )}
            </button>
            {profileOpen && (
              <div className="absolute right-0 mt-2 w-52 bg-panel border border-border rounded-xl shadow-lg overflow-hidden animate-fade-in">
                <div className="px-3.5 py-3 border-b border-border">
                  <p className="text-sm font-semibold truncate">{user?.username}</p>
                  <p className="text-xs text-muted truncate">{user?.email}</p>
                </div>
                <Link
                  href="/account"
                  onClick={() => setProfileOpen(false)}
                  className="flex items-center gap-2.5 w-full px-3.5 py-2.5 text-sm font-medium text-muted hover:text-text hover:bg-accent-soft transition-colors"
                >
                  <UserCircle size={15} />
                  My account
                </Link>
                <button
                  onClick={logout}
                  className="flex items-center gap-2.5 w-full px-3.5 py-2.5 text-sm font-medium text-accent hover:bg-accent-soft transition-colors"
                >
                  <LogOut size={15} />
                  Log out
                </button>
                {version && (
                  <p className="px-3.5 py-2 text-[10.5px] text-muted border-t border-border">IdpForge {version}</p>
                )}
              </div>
            )}
          </div>
        </div>
      </header>

      <main className="flex-1 p-5 md:p-8 max-w-[1200px] w-full mx-auto">
        {user && <CurrentUserProvider value={{ user, permissions }}>{children}</CurrentUserProvider>}
      </main>
    </div>
  );
}
