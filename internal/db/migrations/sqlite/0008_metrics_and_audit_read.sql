CREATE TABLE metrics_snapshots (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	timestamp DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	http_requests INTEGER NOT NULL,
	login_success INTEGER NOT NULL,
	login_failure INTEGER NOT NULL,
	rate_limit_rejections INTEGER NOT NULL
);

CREATE INDEX idx_metrics_snapshots_timestamp ON metrics_snapshots (timestamp);
