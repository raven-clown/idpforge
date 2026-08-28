CREATE TABLE metrics_snapshots (
	id BIGINT AUTO_INCREMENT PRIMARY KEY,
	`timestamp` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	http_requests BIGINT NOT NULL,
	login_success BIGINT NOT NULL,
	login_failure BIGINT NOT NULL,
	rate_limit_rejections BIGINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_metrics_snapshots_timestamp ON metrics_snapshots (`timestamp`);
