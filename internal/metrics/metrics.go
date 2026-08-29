// Package metrics defines IdpForge's custom Prometheus collectors, on top
// of the default Go runtime metrics promhttp.Handler already exposes.
package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "idpforge_http_requests_total",
			Help: "HTTP requests by method, route, and status code.",
		},
		[]string{"method", "route", "status"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "idpforge_http_request_duration_seconds",
			Help:    "HTTP request latency by method and route.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	LoginAttemptsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "idpforge_login_attempts_total",
			Help: "Login attempts by outcome (success, bad_password, mfa_failed, not_found).",
		},
		[]string{"outcome"},
	)

	AuditQueueDepth = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "idpforge_audit_queue_depth",
			Help: "Current number of buffered audit entries awaiting batch insert.",
		},
	)

	RateLimitRejectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "idpforge_rate_limit_rejections_total",
			Help: "Requests rejected by rate limiting, by route.",
		},
		[]string{"route"},
	)

	WebSocketConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "idpforge_websocket_connections",
			Help: "Currently open realtime WebSocket connections on this instance.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		LoginAttemptsTotal,
		AuditQueueDepth,
		RateLimitRejectionsTotal,
		WebSocketConnections,
	)
}
