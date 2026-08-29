CREATE TABLE announcements (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	message TEXT NOT NULL,
	level VARCHAR(16) NOT NULL DEFAULT 'info',
	created_by VARCHAR(64),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_announcements_created_at ON announcements (created_at);
