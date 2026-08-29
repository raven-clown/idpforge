ALTER TABLE api_clients ADD COLUMN folder TEXT;
ALTER TABLE iot_devices ADD COLUMN folder TEXT;

CREATE INDEX idx_api_clients_folder ON api_clients (folder);
CREATE INDEX idx_iot_devices_folder ON iot_devices (folder);
