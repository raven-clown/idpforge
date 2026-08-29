CREATE TABLE leader_leases (
	job_name TEXT PRIMARY KEY,
	holder_id TEXT NOT NULL,
	expires_at DATETIME NOT NULL
);
