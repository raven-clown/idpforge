CREATE TABLE leader_leases (
	job_name VARCHAR(64) PRIMARY KEY,
	holder_id VARCHAR(64) NOT NULL,
	expires_at DATETIME2 NOT NULL
);
