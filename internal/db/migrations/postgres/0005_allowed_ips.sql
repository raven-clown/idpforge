ALTER TABLE api_clients ADD COLUMN allowed_ips JSONB NOT NULL DEFAULT '[]';
ALTER TABLE iot_devices ADD COLUMN allowed_ips JSONB NOT NULL DEFAULT '[]';
