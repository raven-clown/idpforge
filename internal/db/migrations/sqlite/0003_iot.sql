CREATE TABLE iot_devices (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	device_type TEXT NOT NULL,
	location TEXT,
	api_key_hash TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE device_credentials (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	credential_type TEXT NOT NULL,
	credential_ref TEXT NOT NULL,
	label TEXT,
	created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	UNIQUE (credential_type, credential_ref)
);

CREATE INDEX idx_device_credentials_user_id ON device_credentials(user_id);

CREATE TABLE device_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	device_id TEXT NOT NULL REFERENCES iot_devices(id) ON DELETE CASCADE,
	user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
	credential_type TEXT,
	credential_ref TEXT,
	event_type TEXT NOT NULL,
	resource TEXT,
	metadata TEXT NOT NULL DEFAULT '{}',
	status TEXT NOT NULL,
	timestamp DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX idx_device_events_user_timestamp ON device_events(user_id, timestamp);
CREATE INDEX idx_device_events_device_timestamp ON device_events(device_id, timestamp);
CREATE INDEX idx_device_events_event_type ON device_events(event_type, timestamp);
