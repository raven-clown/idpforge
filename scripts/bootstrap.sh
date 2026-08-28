#!/usr/bin/env bash
# Idempotent local/server setup: creates config, data, and log directories
# and a .env file if missing. Safe to run more than once.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_DIR="${IDPFORGE_CONFIG_DIR:-/etc/idpforge}"
DATA_DIR="${IDPFORGE_DATA_DIR:-/var/lib/idpforge}"
LOG_DIR="${IDPFORGE_LOG_DIR:-/var/log/idpforge}"

for dir in "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"; do
	if [ ! -d "$dir" ]; then
		echo "Creating $dir"
		mkdir -p "$dir"
	fi
done

ENV_FILE="$CONFIG_DIR/idpforge.env"
if [ ! -f "$ENV_FILE" ]; then
	echo "Writing $ENV_FILE from .env.example"
	cp "$ROOT_DIR/.env.example" "$ENV_FILE"
	echo "Edit $ENV_FILE before starting the service (set IDPFORGE_DB_DSN at minimum)."
else
	echo "$ENV_FILE already exists, leaving it as is."
fi

echo "Bootstrap complete."
echo "Next steps:"
echo "  1. Edit $ENV_FILE"
echo "  2. Install the systemd unit: sudo cp deploy/systemd/idpforge.service /etc/systemd/system/"
echo "  3. sudo systemctl daemon-reload && sudo systemctl enable --now idpforge"
