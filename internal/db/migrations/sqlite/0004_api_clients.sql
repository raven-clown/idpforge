CREATE TABLE api_clients (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	api_key_hash TEXT NOT NULL,
	allowed_fields TEXT NOT NULL DEFAULT '["id","username"]',
	rate_limit_max INTEGER NOT NULL DEFAULT 60,
	rate_limit_window_seconds INTEGER NOT NULL DEFAULT 60,
	enabled INTEGER NOT NULL DEFAULT 1,
	created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
