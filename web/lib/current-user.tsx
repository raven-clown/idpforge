"use client";

import { createContext, useContext } from "react";
import { User } from "./api";
import { hasPermission } from "./permissions";

// AppShell (the (app) route group's layout) already fetches /api/v1/me
// once per session and owns the auth redirect logic. Pages that just need
// the current user or their permissions should read them from here
// instead of calling useMe() again -- a second call would mean a second
// fetch and a second, redundant redirect check on every page.
interface CurrentUserValue {
  user: User | null;
  permissions: string[];
}

const CurrentUserContext = createContext<CurrentUserValue>({ user: null, permissions: [] });

export const CurrentUserProvider = CurrentUserContext.Provider;

export function useCurrentUser(): User | null {
  return useContext(CurrentUserContext).user;
}

// useCan reports whether the signed-in user holds resource:action, for
// gating an action button (Create, Delete, Revoke, ...) so it's hidden
// rather than shown-then-403'd. The API remains the actual enforcement
// point regardless -- this only controls what the UI offers to click.
export function useCan(resource: string, action: string): boolean {
  const { permissions } = useContext(CurrentUserContext);
  return hasPermission(permissions, resource, action);
}

export function usePermissions(): string[] {
  return useContext(CurrentUserContext).permissions;
}
