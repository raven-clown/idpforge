CREATE TABLE iot_devices (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name VARCHAR(255) NOT NULL UNIQUE,
	device_type VARCHAR(64) NOT NULL,
	location VARCHAR(255),
	api_key_hash VARCHAR(255) NOT NULL,
	enabled BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- credential_ref is an external reference (card number, device-side
-- biometric template ID) supplied by the reader hardware. Raw biometric
-- images/templates never reach this server; matching happens on-device,
-- same principle as WebAuthn.
CREATE TABLE device_credentials (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	credential_type VARCHAR(32) NOT NULL,
	credential_ref VARCHAR(255) NOT NULL,
	label VARCHAR(255),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (credential_type, credential_ref)
);

CREATE INDEX idx_device_credentials_user_id ON device_credentials(user_id);

CREATE TABLE device_events (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	device_id UUID NOT NULL REFERENCES iot_devices(id) ON DELETE CASCADE,
	user_id UUID REFERENCES users(id) ON DELETE SET NULL,
	credential_type VARCHAR(32),
	credential_ref VARCHAR(255),
	event_type VARCHAR(64) NOT NULL,
	resource VARCHAR(255),
	metadata JSONB NOT NULL DEFAULT '{}',
	status VARCHAR(32) NOT NULL,
	"timestamp" TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_device_events_user_timestamp ON device_events(user_id, "timestamp");
CREATE INDEX idx_device_events_device_timestamp ON device_events(device_id, "timestamp");
CREATE INDEX idx_device_events_event_type ON device_events(event_type, "timestamp");
