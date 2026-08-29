import AppShell from "@/components/AppShell";

// This layout persists across every /users, /roles, /iot, ... navigation --
// only `children` swaps. AppShell (topbar, session check, WebSocket
// connection) mounts once instead of on every click, which is what
// actually fixes the full-page flash/flicker on menu switches: before this
// existed, each page called <AppShell> itself, so React tore the whole
// shell down and rebuilt it (re-fetching /api/v1/me, reconnecting the
// WebSocket, briefly showing "Loading...") on every single navigation.
export default function AppGroupLayout({ children }: LayoutProps<"/">) {
  return <AppShell>{children}</AppShell>;
}
