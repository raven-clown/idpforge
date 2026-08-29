CREATE TABLE announcements (
	id BIGINT AUTO_INCREMENT PRIMARY KEY,
	message TEXT NOT NULL,
	level VARCHAR(16) NOT NULL DEFAULT 'info',
	created_by VARCHAR(64),
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_announcements_created_at ON announcements (created_at);
