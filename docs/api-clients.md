# API clients

Two ways for an external app, internal tool, or AI/automation service to
call IdpForge without a human login.

## Scoped tokens (`/api/v1`, GitHub PAT style)

Grant a client explicit `resource:action` scopes, the same format as the
roles/permissions system (including a `resource/*` wildcard). The token can
then call the real admin API, restricted to what it was granted. An admin
can only grant a scope they themselves hold, so a token can never come back
with more power than the person who issued it.

Create one:

```bash
curl -X POST https://sso.example.com/api/v1/api-clients \
  -H "Cookie: idpforge_session=..." \
  -d '{
    "name": "my-ai-assistant",
    "scopes": ["users:read", "iot:read"],
    "rate_limit_max": 100,
    "rate_limit_window_seconds": 60
  }'
```

Use it:

```bash
curl https://sso.example.com/api/v1/users \
  -H "X-API-Key: apik_..."
```

Any `/api/v1` route works the same way for a token as it does for a logged
in admin; `requirePermission` checks the token's scopes instead of a
resolved user's RBAC.

### Available scopes

A scope is `resource:action`. The resource names match the `/api/v1` route
groups exactly, and each has a `read` action, a coarser `manage` action
(covers create/update/delete), or both:

| Resource | Actions | Covers |
|---|---|---|
| `users` | `read`, `manage` | list/get users (`read`); create/update/delete/offboard/device-credentials (`manage`) |
| `rbac` | `manage` | roles, permissions, groups, and all role/group/user assignment endpoints |
| `iot` | `read`, `manage` | query device event history (`read`); register devices (`manage`) |
| `api_clients` | `manage` | create/list/delete other API clients |

These are plain strings checked by `requirePermission`, not rows that have
to exist anywhere first, so any of the combinations above just work.
`resource/*` wildcards (e.g. `users/*:read`) work too but only matter once
your own code checks a custom resource name (through `/forwardauth`, for
example) rather than these fixed route groups.

### Scope presets

| Use case | Scopes | Notes |
|---|---|---|
| AI assistant, read-only | `["users:read", "iot:read"]` | Can look up users and query IoT event history. Cannot create/edit/delete anything. |
| AI assistant, can manage users | `["users:read", "users:manage"]` | Add `users:manage` only if the assistant should create/update/offboard users on your behalf. |
| Full admin token | `["users:manage", "rbac:manage", "iot:manage", "api_clients:manage"]` | Equivalent to what a human admin can do. Grant sparingly. |
| Canteen kiosk / door controller | none (uses `X-Device-Key`, not this token type) | See [integration-matrix.md](integration-matrix.md); hardware check-ins go through `/iot/checkin`, not `/api/v1`. |
| Read-only auditing tool | `["users:read", "iot:read", "api_clients:manage"]` | `api_clients:manage` is required to list clients since there's no separate read-only action for that resource. |

## Simple field-filtered access (`/external/v1`, no scopes)

For a client that only needs "verify this login" or "look up a few
fields", skip scopes entirely and use `allowed_fields` instead:

```bash
curl -X POST https://sso.example.com/api/v1/api-clients \
  -H "Cookie: idpforge_session=..." \
  -d '{"name": "internal-wiki", "allowed_fields": ["id", "username", "email"]}'
```

```bash
curl -X POST https://sso.example.com/external/v1/login \
  -H "X-API-Key: apik_..." \
  -d '{"username": "alice", "password": "..."}'
```

The response only ever contains the fields listed in `allowed_fields`,
regardless of what the underlying user record holds.

## Shared controls

Both paths support, per client:

- `allowed_ips`: CIDR/IP allowlist. Empty means unrestricted.
- `rate_limit_max` / `rate_limit_window_seconds`: independent of the
  global per-IP/per-user rate limit.

Revoke a client with `DELETE /api/v1/api-clients/:id`; the key stops
working immediately, nothing to rotate on the caller's side.
