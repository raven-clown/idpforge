CREATE TABLE announcements (
	id BIGINT IDENTITY(1,1) PRIMARY KEY,
	message NVARCHAR(MAX) NOT NULL,
	level VARCHAR(16) NOT NULL DEFAULT 'info',
	created_by VARCHAR(64),
	created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
);

CREATE INDEX idx_announcements_created_at ON announcements (created_at);
