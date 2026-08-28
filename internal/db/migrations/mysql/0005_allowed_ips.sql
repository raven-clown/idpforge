ALTER TABLE api_clients ADD COLUMN allowed_ips JSON NULL;
ALTER TABLE iot_devices ADD COLUMN allowed_ips JSON NULL;
