CREATE TABLE metrics_snapshots (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	"timestamp" TIMESTAMPTZ NOT NULL DEFAULT now(),
	http_requests BIGINT NOT NULL,
	login_success BIGINT NOT NULL,
	login_failure BIGINT NOT NULL,
	rate_limit_rejections BIGINT NOT NULL
);

CREATE INDEX idx_metrics_snapshots_timestamp ON metrics_snapshots ("timestamp");
