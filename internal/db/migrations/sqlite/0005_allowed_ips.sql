ALTER TABLE api_clients ADD COLUMN allowed_ips TEXT NOT NULL DEFAULT '[]';
ALTER TABLE iot_devices ADD COLUMN allowed_ips TEXT NOT NULL DEFAULT '[]';
