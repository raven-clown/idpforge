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

### Scope presets

| Use case | Scopes | Notes |
|---|---|---|
| AI assistant, general read access | `["users:read", "iot:read"]` | Can look up users and query IoT event history. Cannot create/edit/delete anything. |
| AI assistant, allowed to manage users | `["users:read", "users:write"]` | Add `"users:write"` only if the assistant should be able to create/update/offboard users on your behalf. |
| Read-only reporting bot | `["users:read", "iot:read", "reports/*:read"]` | `reports/*` covers every resource under that folder with one grant. |
| Canteen kiosk / door controller | none (uses `X-Device-Key`, not this token type) | See [integration-matrix.md](integration-matrix.md); hardware check-ins go through `/iot/checkin`, not `/api/v1`. |
| CI/deploy automation | `["api_clients:read"]` | Least-privilege example: can list clients for auditing, nothing else. |

`users:write` is not a real granted-by-default scope; permissions must
exist in the `permissions` table (`rbac_handlers.go` / `POST
/api/v1/rbac/permissions`) before they can be granted to a role or an API
client. Create the ones you need once, then reuse them.

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
