#!/usr/bin/env bash
# Interactively registers a new OIDC application in the applications table.
# Requires psql on PATH and IDPFORGE_DB_DSN pointing at Postgres; for other
# drivers this prints the INSERT for you to run with your own client.
set -euo pipefail

read -rp "Application name: " APP_NAME
read -rp "Redirect URI(s), comma-separated: " REDIRECT_URIS
CLIENT_ID="$(uuidgen 2>/dev/null || cat /proc/sys/kernel/random/uuid)"
CLIENT_SECRET="$(openssl rand -hex 32)"
CLIENT_SECRET_HASH="$(printf '%s' "$CLIENT_SECRET" | openssl dgst -sha256 | awk '{print $2}')"

IFS=',' read -ra URI_ARRAY <<<"$REDIRECT_URIS"
URI_JSON=$(printf '"%s",' "${URI_ARRAY[@]}")
URI_JSON="[${URI_JSON%,}]"

CONFIG_JSON=$(cat <<JSON
{"client_id":"$CLIENT_ID","client_secret_hash":"$CLIENT_SECRET_HASH","redirect_uris":$URI_JSON,"allowed_scopes":["openid","profile","email"]}
JSON
)

SQL="INSERT INTO applications (id, name, protocol, config) VALUES (gen_random_uuid(), '$APP_NAME', 'oidc', '$CONFIG_JSON');"

echo
echo "client_id:     $CLIENT_ID"
echo "client_secret: $CLIENT_SECRET   (save this now, it is not stored in plaintext)"
echo

if [ -n "${IDPFORGE_DB_DRIVER:-}" ] && [ "$IDPFORGE_DB_DRIVER" != "postgres" ]; then
	echo "IDPFORGE_DB_DRIVER=$IDPFORGE_DB_DRIVER is not auto-applied by this script."
	echo "Run the following against your database (adjust UUID generation for your dialect):"
	echo "$SQL"
	exit 0
fi

if command -v psql >/dev/null 2>&1 && [ -n "${IDPFORGE_DB_DSN:-}" ]; then
	psql "$IDPFORGE_DB_DSN" -c "$SQL"
	echo "Application registered."
else
	echo "psql not found or IDPFORGE_DB_DSN not set. Run this SQL manually:"
	echo "$SQL"
fi
