CREATE TABLE metrics_snapshots (
	id BIGINT IDENTITY(1,1) PRIMARY KEY,
	[timestamp] DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
	http_requests BIGINT NOT NULL,
	login_success BIGINT NOT NULL,
	login_failure BIGINT NOT NULL,
	rate_limit_rejections BIGINT NOT NULL
);

CREATE INDEX idx_metrics_snapshots_timestamp ON metrics_snapshots ([timestamp]);
