# IdpForge

Self-hosted SSO / identity platform written in Go, with a Next.js admin
console built as a static export and embedded into the binary via
`go:embed`. Ships as one static executable, no cgo, no Node.js at runtime;
runs the same way on Linux and Windows as a native service or a container,
and scales from a single SQLite file to a multi-instance cluster behind
Postgres/MySQL/MSSQL and Redis without changing anything but config.

> Screenshots aren't checked in yet -- drop your own into
> `docs/screenshots/` and link them here. The walkthrough below describes
> exactly what you'll see.

## What it looks like

Sign in and land on a topbar-driven console (icon nav with an active-tab
indicator, live search, a notification bell, light/dark theme) -- not a
sidebar-and-whitespace admin scaffold. Every section below is real, backed
by the same JSON API a script or another app would call:

- **Dashboard** -- at-a-glance counts (users, roles, API clients, IoT
  devices).
- **Users** -- create, offboard, delete; assign roles; enroll device
  credentials (card/face/fingerprint reference, never raw biometric data);
  reset a password back to the shared default. Every list an admin might
  need to scroll through hundreds of rows in (users, audit log, IoT
  events) is paginated.
- **Roles & permissions** -- roles, permissions, groups (with hierarchy),
  and the grant/revoke UI between them.
- **API clients** -- scoped tokens (GitHub-PAT style): pick scopes from a
  checkbox grid instead of typing `resource:action` strings from memory,
  organize related clients into folders, get usage examples (`curl`
  snippets) the moment a key is issued.
- **IoT devices** -- register a reader, see recent check-in events, and
  read an in-app explanation of exactly how the integration contract
  works (what the device sends, what it gets back, how a new credential
  gets enrolled).
- **Usage** -- request/login graphs and storage usage over time, sampled
  every 10 minutes, no Prometheus required (though it's there too).
- **Audit log** -- filterable, and updates live over WebSocket as new
  entries happen.
- **Settings** -- read-only view of the running config, including the
  org-wide default password and password policy (visible here and in
  server config only, nowhere else).
- **My account** -- self-service avatar upload, TOTP MFA enrollment, and
  WebAuthn security key registration.

The whole UI is permission-aware: `/api/v1/me` returns the signed-in
user's resolved permissions, and the nav/action buttons hide whatever
that user isn't allowed to do, instead of showing it and letting the API
reject it after the fact.

## Why you might use this

- **One binary, your choice of database.** Point `IDPFORGE_DB_DSN` at
  Postgres, MySQL/MariaDB, MSSQL, a managed service (Supabase, RDS, Azure
  SQL, ...), or just use SQLite for a single-node setup. Migrations apply
  automatically on startup.
- **Actually clusters.** Turn on Redis and run more than one instance:
  sessions and the RBAC permission cache are shared, background jobs
  (update checker, health alerts, metrics sampler) elect a leader via a
  DB-backed lease instead of tripling their own work, and the realtime
  WebSocket feed fans out across instances via Redis pub/sub.
- **Onboarding that can't leak a chosen password.** Every new account
  gets one server-configured default password and is forced to change it
  on first login. No API anywhere accepts a caller-chosen password for
  someone else, and none ever returns a password value.
- **Talks to your other apps.** A real OIDC provider (`authorization_code`
  + mandatory PKCE, `refresh_token`, `client_credentials` for
  service-to-service auth, JWKS, discovery) for anything that speaks
  OpenID Connect -- GitLab, Harbor, Rancher, Grafana, Jenkins, Vault,
  NiFi, MinIO, and so on -- plus a forward-auth endpoint for legacy apps
  behind Traefik that don't speak SSO at all.
- **Automatable.** Scoped API-client tokens (grant exactly the
  `resource:action` scopes a script or AI assistant needs), a simpler
  field-filtered `/external/v1` path for apps that just need "verify a
  login" or "provision a user," and an IoT check-in API for physical
  access control (badge/face/fingerprint readers) where matching happens
  on the device and this server only ever sees an opaque reference.
- **Tells you when something's wrong.** Health-check state changes and
  new-version notices show up as in-app announcements in the same
  notification bell everyone already sees, not buried in a log file.

## Quickstart (Docker Compose)

```bash
cd docker
docker compose up --build
```

This starts Postgres, Redis, and the server on `http://localhost:8080`.

## Building from source

The admin console (`web/`, Next.js) is built separately and copied into
`internal/webui/dist/`, where `go:embed` picks it up; `make build` and
`build.ps1` do this automatically. A minimal placeholder is committed
there so a bare `go build` still produces a working binary if you skip
the web build, just without the real admin UI.

```bash
make build             # builds web/ then the Go binary for your platform
```

`build.ps1` always builds `web/` first, then cross-compiles every release
target (see below).

To build the Go binary only, without touching `web/`:

```bash
go build -o dist/idpforge-server ./cmd/server
```

Cross-compile every release target:

```bash
make release          # Linux/macOS
./build.ps1            # Windows
```

Produces `dist/idpforge-server-<os>-<arch>[.exe]` and matching `.tar.gz`
archives for `linux/amd64`, `linux/arm64`, `windows/amd64`, `windows/arm64`.

## Configuration

Copy `.env.example`, fill in `IDPFORGE_DB_DSN` at minimum. Notable knobs
beyond the database connection:

| Variable | What it controls |
|---|---|
| `IDPFORGE_DEFAULT_PASSWORD` | The one password every new/reset account gets, forced to change on first login |
| `IDPFORGE_PASSWORD_MIN_LENGTH`, `IDPFORGE_PASSWORD_REQUIRE_*` | Complexity policy for self-chosen passwords |
| `IDPFORGE_ACCOUNT_LOCKOUT_MAX_ATTEMPTS`, `_WINDOW`, `_DURATION` | Per-account login lockout, independent of the per-IP rate limit |
| `IDPFORGE_REDIS_ENABLED` | Session sharing, RBAC cache sharing, and cross-instance realtime -- turn this on to actually cluster |
| `IDPFORGE_TIMEZONE` | IdpForge's own reference zone for schedules/logs; timestamps in the UI always render in each viewer's own browser zone regardless |
| `IDPFORGE_UPDATE_CHECK_ENABLED` | Poll GitHub Releases and post an in-app notice when a newer version is out (never auto-updates) |

See [docs/platform-support.md](docs/platform-support.md) for the full
variable list and [docs/runbook.md](docs/runbook.md) for operations.

**A note on SQLite**: it's the zero-config default for trying IdpForge out,
but it's a single local file -- no real concurrent-write support and no
way to share it across multiple instances. Use Postgres, MySQL, or MSSQL
for anything beyond a single small-team, single-instance deployment.

## Deploying as a service

- Linux: `deploy/systemd/idpforge.service`
- Windows: `deploy/windows/install-service.ps1`

Both `scripts/bootstrap.sh` and `scripts/bootstrap.ps1` set up config/data/
log directories idempotently.

## Registering an OIDC application

```bash
scripts/add-app.sh      # Linux/macOS
scripts/add-app.ps1      # Windows
```

## Docker images

Published to both Docker Hub and GitHub Container Registry on every
`vX.Y.Z` tag:

```
docker.io/<namespace>/idpforge:<version>
docker.io/<namespace>/idpforge:latest

ghcr.io/raven-clown/idpforge:<version>
ghcr.io/raven-clown/idpforge:latest
```

Multi-arch (`linux/amd64`, `linux/arm64`), built and pushed by
`.github/workflows/release.yml`. The GHCR push needs no extra secrets --
just the token GitHub Actions already provides for the repo.

## Integration matrix

See [docs/integration-matrix.md](docs/integration-matrix.md).

## API clients (scoped tokens, AI/automation access)

See [docs/api-clients.md](docs/api-clients.md) for scope presets,
`/external/v1` examples (including provisioning a user), and the full
`resource:action` reference.

## Monitoring

See [docs/monitoring.md](docs/monitoring.md): a Prometheus scrape config
and Grafana dashboard are in `deploy/monitoring/`, and the admin console
has its own built-in usage graphs (`/usage`) that don't need either.

## Contributing

Issues and PRs welcome, including from first-time contributors. See
[CONTRIBUTING.md](CONTRIBUTING.md) for local setup, code conventions,
and what to check before opening a PR.

Not sure where to start? Browse the
[`good first issue`](https://github.com/raven-clown/idpforge/issues?q=is%3Aopen+label%3A%22good+first+issue%22)
label for small, well-scoped tasks, or
[`help wanted`](https://github.com/raven-clown/idpforge/issues?q=is%3Aopen+label%3A%22help+wanted%22)
for bigger ones. Both are seeded from the project's own honest gap list,
see [CHANGELOG.md](CHANGELOG.md)'s "Known limitations" section for the
full picture of what's shipped and what isn't yet.

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).

## License

MIT, see [LICENSE](LICENSE).
