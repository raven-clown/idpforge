ALTER TABLE users ADD COLUMN password_changed_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'));
