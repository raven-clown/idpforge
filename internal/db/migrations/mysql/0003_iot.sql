CREATE TABLE iot_devices (
	id CHAR(36) PRIMARY KEY,
	name VARCHAR(255) NOT NULL UNIQUE,
	device_type VARCHAR(64) NOT NULL,
	location VARCHAR(255),
	api_key_hash VARCHAR(255) NOT NULL,
	enabled BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE device_credentials (
	id CHAR(36) PRIMARY KEY,
	user_id CHAR(36) NOT NULL,
	credential_type VARCHAR(32) NOT NULL,
	credential_ref VARCHAR(255) NOT NULL,
	label VARCHAR(255),
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE (credential_type, credential_ref),
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_device_credentials_user_id ON device_credentials(user_id);

CREATE TABLE device_events (
	id BIGINT AUTO_INCREMENT PRIMARY KEY,
	device_id CHAR(36) NOT NULL,
	user_id CHAR(36),
	credential_type VARCHAR(32),
	credential_ref VARCHAR(255),
	event_type VARCHAR(64) NOT NULL,
	resource VARCHAR(255),
	metadata JSON NOT NULL,
	status VARCHAR(32) NOT NULL,
	`timestamp` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (device_id) REFERENCES iot_devices(id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_device_events_user_timestamp ON device_events(user_id, `timestamp`);
CREATE INDEX idx_device_events_device_timestamp ON device_events(device_id, `timestamp`);
CREATE INDEX idx_device_events_event_type ON device_events(event_type, `timestamp`);
