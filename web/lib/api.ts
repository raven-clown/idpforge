// Thin fetch wrapper around the Go backend's JSON API. The static-exported
// Next.js app is served by the same Go binary, so relative paths (/api/v1/*)
// resolve against whatever origin served this page: no base URL needed.

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    credentials: "include",
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
  });
  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = await res.json();
      message = body.message || body.error || message;
    } catch {
      // ignore, non-JSON error body
    }
    throw new ApiError(res.status, message);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

const get = <T,>(path: string) => request<T>(path);
const post = <T,>(path: string, body?: unknown) =>
  request<T>(path, { method: "POST", body: body ? JSON.stringify(body) : undefined });
const patch = <T,>(path: string, body?: unknown) =>
  request<T>(path, { method: "PATCH", body: body ? JSON.stringify(body) : undefined });
const del = <T,>(path: string) => request<T>(path, { method: "DELETE" });

export interface User {
  id: string;
  username: string;
  email: string;
  status: "active" | "suspended" | "disabled";
  mfa_enabled: boolean;
  source: string;
  external_id?: string;
  avatar_url?: string;
  created_at: string;
  updated_at: string;
  last_login_at?: string;
}

export interface Role {
  id: string;
  name: string;
  description?: string;
}

export interface PermissionRecord {
  id: string;
  resource: string;
  action: string;
}

export interface Group {
  id: string;
  name: string;
  parent_group_id?: string;
}

export interface ApiClient {
  id: string;
  name: string;
  allowed_fields: string[];
  scopes?: string[];
  allowed_ips?: string[];
  rate_limit_max: number;
  rate_limit_window_seconds: number;
  enabled: boolean;
  created_at: string;
}

export interface Device {
  id: string;
  name: string;
  device_type: string;
  location?: string;
  allowed_ips?: string[];
  enabled: boolean;
  created_at: string;
}

export interface DeviceCredential {
  id: string;
  user_id: string;
  credential_type: string;
  credential_ref: string;
  label?: string;
  created_at: string;
}

export interface DeviceEvent {
  id: number;
  device_id: string;
  user_id?: string;
  event_type: string;
  resource?: string;
  status: string;
  timestamp: string;
}

export interface AuditEntry {
  id: number;
  actor_id?: string;
  actor_ip?: string;
  action: string;
  target_resource?: string;
  target_app?: string;
  status: string;
  timestamp: string;
}

export interface MetricsSnapshot {
  timestamp: string;
  http_requests: number;
  login_success: number;
  login_failure: number;
  rate_limit_rejections: number;
}

export interface Settings {
  env: string;
  http: { listen_addr: string; base_url: string };
  database: { driver: string; dsn: string };
  redis: { enabled: boolean; addr: string };
  rate_limit: {
    enabled: boolean;
    max: number;
    window_seconds: number;
    login_max: number;
    login_window_seconds: number;
  };
  captcha: { provider: string };
  oidc: {
    issuer: string;
    access_token_ttl_minutes: number;
    id_token_ttl_minutes: number;
    refresh_token_ttl_hours: number;
  };
  backup: { enabled: boolean; dir: string; schedule: string; retention_days: number };
  storage: { backend: string };
  password_expiry_days: number;
}

export const api = {
  login: (username: string, password: string, mfa_code: string, captcha_token = "") =>
    post<{ user_id: string; mfa_required: boolean; password_change_required: boolean }>(
      "/api/v1/login",
      { username, password, mfa_code, captcha_token }
    ),
  logout: () => post<{ status: string }>("/api/v1/logout"),

  users: {
    list: () => get<{ users: User[] }>("/api/v1/users?limit=500"),
    get: (id: string) => get<User>(`/api/v1/users/${id}`),
    create: (username: string, email: string, password: string) =>
      post<User>("/api/v1/users", { username, email, password }),
    update: (id: string, body: { email?: string; status?: string }) =>
      patch<User>(`/api/v1/users/${id}`, body),
    remove: (id: string) => del<void>(`/api/v1/users/${id}`),
    offboard: (id: string) => post<User>(`/api/v1/users/${id}/offboard`),
    credentials: (id: string) =>
      get<{ credentials: DeviceCredential[] }>(`/api/v1/users/${id}/device-credentials`),
    addCredential: (id: string, credential_type: string, credential_ref: string, label: string) =>
      post<DeviceCredential>(`/api/v1/users/${id}/device-credentials`, {
        credential_type,
        credential_ref,
        label,
      }),
    removeCredential: (id: string, credId: string) =>
      del<void>(`/api/v1/users/${id}/device-credentials/${credId}`),
  },

  rbac: {
    roles: () => get<{ roles: Role[] }>("/api/v1/rbac/roles"),
    getRole: (id: string) => get<Role>(`/api/v1/rbac/roles/${id}`),
    rolePermissions: (id: string) =>
      get<{ permissions: PermissionRecord[] }>(`/api/v1/rbac/roles/${id}/permissions`),
    createRole: (name: string, description: string) =>
      post<Role>("/api/v1/rbac/roles", { name, description }),
    deleteRole: (id: string) => del<void>(`/api/v1/rbac/roles/${id}`),
    permissions: () => get<{ permissions: PermissionRecord[] }>("/api/v1/rbac/permissions"),
    createPermission: (resource: string, action: string) =>
      post<PermissionRecord>("/api/v1/rbac/permissions", { resource, action }),
    grantPermission: (roleId: string, permissionId: string) =>
      post<void>(`/api/v1/rbac/roles/${roleId}/permissions`, { permission_id: permissionId }),
    revokePermission: (roleId: string, permissionId: string) =>
      del<void>(`/api/v1/rbac/roles/${roleId}/permissions/${permissionId}`),
    userRoles: (userId: string) => get<{ roles: Role[] }>(`/api/v1/rbac/users/${userId}/roles`),
    groups: () => get<{ groups: Group[] }>("/api/v1/rbac/groups"),
    createGroup: (name: string, parentGroupId: string) =>
      post<Group>("/api/v1/rbac/groups", { name, parent_group_id: parentGroupId || undefined }),
    assignRoleToUser: (userId: string, roleId: string) =>
      post<void>(`/api/v1/rbac/users/${userId}/roles`, { role_id: roleId }),
    removeRoleFromUser: (userId: string, roleId: string) =>
      del<void>(`/api/v1/rbac/users/${userId}/roles/${roleId}`),
  },

  apiClients: {
    list: () => get<{ clients: ApiClient[] }>("/api/v1/api-clients"),
    create: (body: {
      name: string;
      scopes: string[];
      allowed_fields: string[];
      allowed_ips: string[];
      rate_limit_max: number;
      rate_limit_window_seconds: number;
    }) => post<{ client: ApiClient; api_key: string }>("/api/v1/api-clients", body),
    remove: (id: string) => del<void>(`/api/v1/api-clients/${id}`),
  },

  iot: {
    devices: () => get<{ devices: Device[] }>("/api/v1/iot/devices"),
    createDevice: (body: { name: string; device_type: string; location: string; allowed_ips: string[] }) =>
      post<{ device: Device; api_key: string }>("/api/v1/iot/devices", body),
    events: (params = "") => get<{ events: DeviceEvent[] }>(`/api/v1/iot/events${params}`),
  },

  auditLogs: (params = "") => get<{ entries: AuditEntry[] }>(`/api/v1/audit-logs${params}`),
  metricsHistory: (days = 30) =>
    get<{ snapshots: MetricsSnapshot[] }>(`/api/v1/metrics/history?days=${days}`),
  settings: () => get<Settings>("/api/v1/settings"),
};
