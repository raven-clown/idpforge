#!/usr/bin/env bash
# Dumps the configured database to IDPFORGE_BACKUP_DIR and prunes backups
# older than IDPFORGE_BACKUP_RETENTION_DAYS. Driver-native dump tools only
# (pg_dump/mysqldump/sqlcmd/sqlite3); this does not manage replication or
# clustering, only point-in-time backups. Schedule with cron or Task
# Scheduler; see docs/runbook.md for a streaming-replication setup instead.
set -euo pipefail

BACKUP_DIR="${IDPFORGE_BACKUP_DIR:-/var/lib/idpforge/backups}"
RETENTION_DAYS="${IDPFORGE_BACKUP_RETENTION_DAYS:-14}"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"

mkdir -p "$BACKUP_DIR"

case "${IDPFORGE_DB_DRIVER:-postgres}" in
postgres)
	OUT="$BACKUP_DIR/idpforge-$TIMESTAMP.dump"
	pg_dump "$IDPFORGE_DB_DSN" -Fc -f "$OUT"
	;;
mysql)
	OUT="$BACKUP_DIR/idpforge-$TIMESTAMP.sql.gz"
	mysqldump --result-file=/dev/stdout "$@" | gzip >"$OUT"
	;;
mssql)
	echo "MSSQL: run BACKUP DATABASE via sqlcmd; not automated here." >&2
	exit 1
	;;
sqlite)
	SRC="${IDPFORGE_DB_DSN#file:}"
	OUT="$BACKUP_DIR/idpforge-$TIMESTAMP.sqlite"
	sqlite3 "$SRC" ".backup '$OUT'"
	;;
*)
	echo "Unknown IDPFORGE_DB_DRIVER" >&2
	exit 1
	;;
esac

echo "Backup written to $OUT"

find "$BACKUP_DIR" -type f -mtime "+$RETENTION_DAYS" -print -delete
