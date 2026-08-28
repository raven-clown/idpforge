# IdpForge

Self-hosted SSO / identity platform written in Go. Single static binary,
no cgo, runs the same way on Linux and Windows as a native service or a
container.

## Features

- Centralized users, groups (with hierarchy), roles, and permissions
- Permission resolution (`user -> group(s) -> role(s) -> permission(s)`)
  cached in Redis (in-memory fallback for single-node setups), invalidated
  on every write
- Async, batched append-only audit log
- OIDC provider (authorization code + PKCE, refresh tokens, JWKS,
  discovery) so apps like GitLab, Harbor, Rancher, Grafana, Jenkins,
  Vault, NiFi, and MinIO can point their native OIDC config at IdpForge
- WebAuthn/FIDO2 (security keys, Windows Hello, Touch ID, fingerprint
  readers) and TOTP for MFA
- Forward-auth endpoint for legacy apps with no SSO support (Traefik)
- IoT/hardware check-in API for badge, face, and fingerprint readers
  (matching happens on the device, this server only sees a credential
  reference), with its own event history separate from the audit log
- Scoped API client tokens (`api-clients`), a GitHub-PAT-style credential:
  grant one exactly the `resource:action` scopes it needs and it can call
  the real admin API, or just a field-filtered read/login check via
  `/external/v1` with no scopes at all
- Pluggable SQL backend: Postgres (incl. Supabase and other managed
  Postgres), MySQL/MariaDB, MSSQL, or SQLite for single-node setups
- Cloudflare Turnstile / hCaptcha on login
- Prometheus metrics, health check
- Runs as a systemd service on Linux and a native Windows Service (no
  NSSM) on Windows

## Quickstart (Docker Compose)

```bash
cd docker
docker compose up --build
```

This starts Postgres, Redis, and the server on `http://localhost:8080`.

## Building from source

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

Copy `.env.example`, fill in `IDPFORGE_DB_DSN` at minimum. See
[docs/platform-support.md](docs/platform-support.md) for the full variable
list and [docs/runbook.md](docs/runbook.md) for operations.

## Deploying as a service

- Linux: `deploy/systemd/idpforge.service`
- Windows: `deploy/windows/install-service.ps1`

Both `scripts/bootstrap.sh` and `scripts/bootstrap.ps1` set up config/data/
log directories idempotently.

## Registering an application

```bash
scripts/add-app.sh      # Linux/macOS
scripts/add-app.ps1      # Windows
```

## Docker images

```
docker.io/<namespace>/idpforge:<version>
docker.io/<namespace>/idpforge:latest
```

Multi-arch (`linux/amd64`, `linux/arm64`) built and pushed by
`.github/workflows/release.yml` on every `vX.Y.Z` tag.

## Integration matrix

See [docs/integration-matrix.md](docs/integration-matrix.md).

## API clients (scoped tokens, AI/automation access)

See [docs/api-clients.md](docs/api-clients.md) for scope presets and
examples.

## License

MIT, see [LICENSE](LICENSE).
