# Runbook

## First run

1. `scripts/bootstrap.sh` or `scripts/bootstrap.ps1`
2. Edit the generated `idpforge.env`, set `IDPFORGE_DB_DSN`
3. Start the service (systemd or Windows Service). Migrations run
   automatically on startup for the configured driver.

## Backups

`scripts/backup.sh` / `scripts/backup.ps1` dump the database with the
driver-native tool (`pg_dump`, `mysqldump`, `sqlite3 .backup`) and prune
anything older than `IDPFORGE_BACKUP_RETENTION_DAYS`. Schedule with cron
or Windows Task Scheduler; there is no built-in scheduler in the server
process itself.

Restore with `scripts/restore.sh <file>` / `scripts/restore.ps1 -BackupFile <file>`.

## Replication / high availability

IdpForge does not implement database clustering or replication itself;
use your database's own mechanism instead:

- Postgres: streaming replication or a managed replica (RDS Multi-AZ,
  Cloud SQL HA, Supabase read replicas)
- MySQL: standard primary/replica replication
- MSSQL: Always On availability groups

Point `IDPFORGE_DB_DSN` at the primary. The server itself is stateless
(all session/RBAC/OIDC-code state lives in Redis or the database), so
running multiple IdpForge instances behind a load balancer against the
same database and Redis is safe as long as Redis is shared, not the
per-node in-memory cache fallback.

## Offboarding a user

`POST /api/v1/users/:id/offboard` disables the account, invalidates its
cached permissions, and writes a single audit entry covering the action.
Downstream per-app deprovisioning (SCIM) is planned; until then, revoke
per-app sessions/tokens via each app's own admin API.

## Rotating the OIDC signing key

Delete the file at `IDPFORGE_OIDC_SIGNING_KEY` (default: OS config dir)
and restart; a new key is generated automatically. All previously issued
tokens become unverifiable, so expect every client to need to
re-authenticate.
