# Platform support matrix

| Feature | Linux | Windows |
|---|---|---|
| Native binary | yes | yes |
| Systemd service | yes | n/a |
| Windows Service | n/a | yes (`x/sys/windows/svc`, no NSSM) |
| Container image | `docker.io/<namespace>/idpforge` (distroless, multi-arch) | Dockerfile.windows provided; not yet in the automated release pipeline |
| CGO | disabled everywhere | disabled everywhere |
| CI | GitHub Actions, `ubuntu-latest` | GitHub Actions, `windows-latest` |

## Environment variables

| Variable | Default | Notes |
|---|---|---|
| `IDPFORGE_ENV` | `development` | `production` disables non-secure session cookies |
| `IDPFORGE_LISTEN_ADDR` | `:8080` | |
| `IDPFORGE_BASE_URL` | `http://localhost:8080` | also the WebAuthn RP origin |
| `IDPFORGE_DB_DRIVER` | `postgres` | `postgres`, `mysql`, `mssql`, `sqlite` |
| `IDPFORGE_DB_DSN` | (required) | connection string for the chosen driver; any Postgres-wire-compatible managed service (Supabase, RDS, Cloud SQL, ...) works with `postgres` |
| `IDPFORGE_DB_MAX_OPEN_CONNS` | `25` | |
| `IDPFORGE_DB_MAX_IDLE_CONNS` | `5` | |
| `IDPFORGE_DB_CONN_MAX_LIFETIME` | `30m` | |
| `IDPFORGE_REDIS_ENABLED` | `true` | `false` uses the in-memory cache (single-node only) |
| `IDPFORGE_REDIS_ADDR` | `localhost:6379` | |
| `IDPFORGE_REDIS_PASSWORD` | (empty) | |
| `IDPFORGE_AUDIT_BATCH_SIZE` | `100` | |
| `IDPFORGE_AUDIT_FLUSH_INTERVAL` | `2s` | |
| `IDPFORGE_AUDIT_QUEUE_SIZE` | `10000` | |
| `IDPFORGE_CAPTCHA_PROVIDER` | `none` | `none`, `turnstile`, `hcaptcha` |
| `IDPFORGE_CAPTCHA_SITE_KEY` | (empty) | |
| `IDPFORGE_CAPTCHA_SECRET_KEY` | (empty) | |
| `IDPFORGE_OIDC_ISSUER` | `http://localhost:8080` | must match what clients expect |
| `IDPFORGE_OIDC_SIGNING_KEY` | OS config dir | generated on first run if missing |
| `IDPFORGE_OIDC_ACCESS_TTL` | `15m` | |
| `IDPFORGE_OIDC_ID_TTL` | `15m` | |
| `IDPFORGE_OIDC_REFRESH_TTL` | `720h` | |
| `IDPFORGE_BACKUP_ENABLED` | `false` | informational; actual backups run via `scripts/backup.sh`/`.ps1` on a schedule |
| `IDPFORGE_BACKUP_DIR` | OS data dir/backups | |
| `IDPFORGE_BACKUP_RETENTION_DAYS` | `14` | |
