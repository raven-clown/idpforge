CREATE TABLE iot_devices (
	id UNIQUEIDENTIFIER PRIMARY KEY,
	name NVARCHAR(255) NOT NULL UNIQUE,
	device_type NVARCHAR(64) NOT NULL,
	location NVARCHAR(255),
	api_key_hash NVARCHAR(255) NOT NULL,
	enabled BIT NOT NULL DEFAULT 1,
	created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
);

CREATE TABLE device_credentials (
	id UNIQUEIDENTIFIER PRIMARY KEY,
	user_id UNIQUEIDENTIFIER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	credential_type NVARCHAR(32) NOT NULL,
	credential_ref NVARCHAR(255) NOT NULL,
	label NVARCHAR(255),
	created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
	CONSTRAINT uq_device_credentials UNIQUE (credential_type, credential_ref)
);

CREATE INDEX idx_device_credentials_user_id ON device_credentials(user_id);

CREATE TABLE device_events (
	id BIGINT IDENTITY(1,1) PRIMARY KEY,
	device_id UNIQUEIDENTIFIER NOT NULL REFERENCES iot_devices(id) ON DELETE CASCADE,
	user_id UNIQUEIDENTIFIER NULL,
	credential_type NVARCHAR(32),
	credential_ref NVARCHAR(255),
	event_type NVARCHAR(64) NOT NULL,
	resource NVARCHAR(255),
	metadata NVARCHAR(MAX) NOT NULL DEFAULT '{}',
	status NVARCHAR(32) NOT NULL,
	[timestamp] DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_device_events_user_timestamp ON device_events(user_id, [timestamp]);
CREATE INDEX idx_device_events_device_timestamp ON device_events(device_id, [timestamp]);
CREATE INDEX idx_device_events_event_type ON device_events(event_type, [timestamp]);
