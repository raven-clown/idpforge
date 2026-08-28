#!/usr/bin/env bash
# Restores a backup produced by backup.sh. Destructive: run against an empty
# or intentionally-overwritten database only.
set -euo pipefail

if [ $# -ne 1 ]; then
	echo "Usage: $0 <backup-file>" >&2
	exit 1
fi
BACKUP_FILE="$1"

case "${IDPFORGE_DB_DRIVER:-postgres}" in
postgres)
	pg_restore -d "$IDPFORGE_DB_DSN" --clean --if-exists "$BACKUP_FILE"
	;;
mysql)
	gunzip -c "$BACKUP_FILE" | mysql "$@"
	;;
sqlite)
	SRC="${IDPFORGE_DB_DSN#file:}"
	cp "$BACKUP_FILE" "$SRC"
	;;
*)
	echo "Restore for this driver is not automated; restore $BACKUP_FILE manually." >&2
	exit 1
	;;
esac

echo "Restore complete."
