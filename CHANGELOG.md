# Changelog

All notable changes to IdpForge are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [1.0.0]

### Security

- **Fixed**: `client_secret` is now actually verified against the stored
  hash for the OIDC `authorization_code` and `refresh_token` grants. Before
  this, a "confidential" client's secret was accepted and silently ignored
  -- anyone who knew a `client_id` could complete the token exchange
  without it.
- Added security response headers (CSP, `X-Frame-Options: DENY`, HSTS in
  non-development environments, and the rest of the standard set) --
  previously there were none at all.
- Added a lightweight CSRF defense: any state-changing request that
  carries the session cookie must present a matching `Origin` (or
  `Referer`), on top of the existing `SameSite=Lax` cookie.
- Added per-account login lockout (`IDPFORGE_ACCOUNT_LOCKOUT_*`),
  independent of the existing per-IP rate limit, which alone did nothing
  against an attacker spreading attempts across many source IPs.
- Added a configurable password complexity policy
  (`IDPFORGE_PASSWORD_REQUIRE_*`, `IDPFORGE_PASSWORD_MIN_LENGTH`) enforced
  on every self-chosen password.

### Added

- **Default-password onboarding**: every new account (via the admin UI,
  `/api/v1/users`, or the new `/external/v1/users`) is created with a
  single server-configured default password
  (`IDPFORGE_DEFAULT_PASSWORD`) and forced to change it on first login.
  An admin can reset any account back to the default the same way. No API
  anywhere accepts a caller-chosen password for another account, and none
  ever returns a password value -- the default is visible only in server
  config or the admin Settings page.
- **Permission-aware admin UI**: `/api/v1/me` now returns the caller's
  resolved permissions, and the topbar/action buttons hide what the
  signed-in user isn't allowed to do, instead of showing it and letting
  the API reject it after the fact.
- **Realtime notifications**: a WebSocket feed pushes new audit log
  entries (to users with `audit:read`) and announcements (to everyone) as
  they happen. Admins can post an announcement to everyone signed in;
  IdpForge itself posts one automatically when a health check changes
  state or a newer release is available.
- **Update checker**: polls GitHub Releases on an interval and posts an
  in-app announcement when a newer version is out. Never auto-updates --
  notification only.
- **Cluster-safe by default**: the update checker, health-alert poller,
  and metrics sampler now use a DB-backed leader lease, so running
  multiple instances no longer triples their work or their announcements.
  With Redis enabled, the realtime WebSocket feed also fans out across
  every instance instead of staying local to whichever one handled the
  request.
- **Folders** for API clients and IoT devices: an optional organizational
  label for the admin UI's list, distinct from (and not a replacement
  for) each entry's own individually-revocable key.
- **Employee ID** field on user accounts (optional, unique when set).
- `POST /external/v1/users`: lets a scoped API client (with the
  `users:manage` scope) provision an account through the same simple path
  used for login/lookup, for an HR or onboarding integration that
  shouldn't need the full admin API.
- A storage-usage graph (avatar disk usage over time) on the Usage page.
- Two more Prometheus metrics and Grafana panels: `idpforge_websocket_connections`
  (live realtime connections on this instance) and an account-lockouts
  panel driven by the existing `idpforge_login_attempts_total{outcome="locked"}`.
- Self-service **My account** page: avatar upload, TOTP MFA
  enroll/disable, and WebAuthn security key registration -- the backend
  endpoints existed already, nothing in the UI called them until now.
- OAuth 2.0 `client_credentials` grant for plain service-to-service auth
  (no user, no browser): a client authenticates with just its own
  `client_id`/`client_secret` and gets a token scoped to exactly its
  `allowed_scopes`, no refresh token issued.
- `internal/httpserver` integration tests (previously zero) covering
  login, lockout, permission enforcement, CSRF, and the default-password
  invariants.

### Changed

- Rebuilt the admin console's layout as a real persistent Next.js layout
  instead of each page mounting its own shell -- navigating between pages
  no longer re-fetches the session, reconnects the WebSocket, or flashes
  a full-page loading state on every click.
- Replaced several free-text fields that only ever took one of a known
  set of values (API client scopes, allowed fields, device/credential
  type) with checkboxes or dropdowns.
- Timestamps render in each viewer's own browser time zone automatically,
  rather than a fixed zone -- what actually matters for staff spread
  across offices.
- Themed the scrollbar and fixed a layout-shift jitter
  (`scrollbar-gutter: stable`) when navigating between a scrollable and a
  non-scrollable page.
- Added pagination to the Users, Audit log, and IoT check-in events lists.

### Fixed

- The `/iot` admin page was completely unreachable: the hardware
  check-in API was mounted at the same path prefix, and Fiber's
  prefix-matched middleware intercepted every browser request to the page
  before it could render. Moved the device API to `/device/v1/checkin`.

### Known limitations

Honest gaps, not hidden ones:

- No self-service "forgot password" flow -- only an admin can reset an
  account (to the shared default). There's no outbound email/SMTP
  integration in IdpForge at all yet, which a real reset-link flow would
  need.
- No OIDC application (relying-party) management API or UI -- `applications`
  rows are still created by hand via `scripts/add-app.sh`/`.ps1`.
- No admin UI to list or revoke a user's registered WebAuthn credentials
  or active sessions (the backend can remove a WebAuthn credential;
  nothing calls it yet).
- No end-employee-facing portal ("here are your apps") -- everything
  built is the admin console. Employees only ever see IdpForge at the
  OIDC login screen from a relying-party app.
- Real Postgres/MySQL/MSSQL migrations have only ever been run through
  the SQLite-backed test helper, never against a live instance of any of
  the three.
- No audit log retention/archival job -- it grows forever.
- No Windows container image in CI/release, despite Windows being a
  build target.
- `metrics-sampler`'s recorded numbers are per-instance (in-process
  atomic counters), not cluster-wide, even though the leader lease
  prevents duplicate rows in a multi-instance deployment.

## [0.1.2] - 2026-08-28

- Replaced the server-rendered (Go `html/template`) admin console with a
  Next.js 16 static export, embedded into the binary via `go:embed` --
  same single-binary distribution model, modern UI.
- Added usage graphs, an audit log viewer, a settings page, and
  Grafana/Prometheus dashboard templates.

## [0.1.1] - 2026-08-28

- Added rate limiting, avatar upload/storage (local disk or S3-compatible),
  IoT device integration (badge/face/fingerprint check-in,
  credential-reference model), scoped API client tokens, and admin
  permission scopes.

## [0.1.0] - 2026-08-28

- Initial scaffold: multi-database support (Postgres, MySQL, MSSQL,
  SQLite), RBAC (users → groups → roles → permissions), async audit
  logging, an OIDC provider (authorization_code + PKCE, refresh tokens,
  JWKS), WebAuthn and TOTP MFA, cross-platform (Linux/Windows) builds.
