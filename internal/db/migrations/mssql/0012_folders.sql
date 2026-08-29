ALTER TABLE api_clients ADD folder VARCHAR(128);
ALTER TABLE iot_devices ADD folder VARCHAR(128);

CREATE INDEX idx_api_clients_folder ON api_clients (folder);
CREATE INDEX idx_iot_devices_folder ON iot_devices (folder);
