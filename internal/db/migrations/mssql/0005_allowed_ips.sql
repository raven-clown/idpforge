ALTER TABLE api_clients ADD allowed_ips NVARCHAR(MAX) NOT NULL DEFAULT '[]';
ALTER TABLE iot_devices ADD allowed_ips NVARCHAR(MAX) NOT NULL DEFAULT '[]';
