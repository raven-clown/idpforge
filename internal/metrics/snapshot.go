package metrics

import "sync/atomic"

// Plain cumulative counters alongside the labeled Prometheus vectors above,
// so a periodic sampler can snapshot a single number per metric into
// metrics_snapshots without parsing the Prometheus registry. Prometheus
// itself remains the source of truth for anyone scraping /metrics; these
// exist only to give the admin UI's usage graphs history to draw from.
var (
	totalRequests          atomic.Int64
	totalLoginSuccess      atomic.Int64
	totalLoginFailure      atomic.Int64
	totalRateLimitRejected atomic.Int64
)

type Totals struct {
	HTTPRequests        int64
	LoginSuccess        int64
	LoginFailure        int64
	RateLimitRejections int64
}

func CurrentTotals() Totals {
	return Totals{
		HTTPRequests:        totalRequests.Load(),
		LoginSuccess:        totalLoginSuccess.Load(),
		LoginFailure:        totalLoginFailure.Load(),
		RateLimitRejections: totalRateLimitRejected.Load(),
	}
}

func RecordHTTPRequest(method, route, status string) {
	HTTPRequestsTotal.WithLabelValues(method, route, status).Inc()
	totalRequests.Add(1)
}

func RecordLoginAttempt(outcome string) {
	LoginAttemptsTotal.WithLabelValues(outcome).Inc()
	if outcome == "success" {
		totalLoginSuccess.Add(1)
	} else {
		totalLoginFailure.Add(1)
	}
}

func RecordRateLimitRejection(route string) {
	RateLimitRejectionsTotal.WithLabelValues(route).Inc()
	totalRateLimitRejected.Add(1)
}
