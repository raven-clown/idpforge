CREATE TABLE announcements (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	message TEXT NOT NULL,
	level TEXT NOT NULL DEFAULT 'info',
	created_by TEXT,
	created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX idx_announcements_created_at ON announcements (created_at);
