package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/raven-clown/idpforge/internal/db"
)

type Snapshot struct {
	Timestamp           time.Time `json:"timestamp"`
	HTTPRequests        int64     `json:"http_requests"`
	LoginSuccess        int64     `json:"login_success"`
	LoginFailure        int64     `json:"login_failure"`
	RateLimitRejections int64     `json:"rate_limit_rejections"`
}

type History struct {
	db *db.DB
}

func NewHistory(database *db.DB) *History {
	return &History{db: database}
}

// Record inserts one row of the current cumulative totals. Called
// periodically by a background sampler (see cmd/server/main.go), not on
// every request.
func (h *History) Record(ctx context.Context, t Totals) error {
	q := fmt.Sprintf(`INSERT INTO metrics_snapshots (http_requests, login_success, login_failure, rate_limit_rejections)
VALUES (%s, %s, %s, %s)`, h.db.Placeholder(1), h.db.Placeholder(2), h.db.Placeholder(3), h.db.Placeholder(4))
	_, err := h.db.ExecContext(ctx, q, t.HTTPRequests, t.LoginSuccess, t.LoginFailure, t.RateLimitRejections)
	return err
}

// Since returns every snapshot from `since` onward, oldest first, so the
// admin UI can plot deltas between consecutive points as a usage-over-time
// chart.
func (h *History) Since(ctx context.Context, since time.Time) ([]Snapshot, error) {
	q := fmt.Sprintf(`SELECT "timestamp", http_requests, login_success, login_failure, rate_limit_rejections
FROM metrics_snapshots WHERE "timestamp" >= %s ORDER BY "timestamp" ASC`, h.db.Placeholder(1))
	if h.db.Driver == "mysql" || h.db.Driver == "sqlite" {
		q = fmt.Sprintf(`SELECT timestamp, http_requests, login_success, login_failure, rate_limit_rejections
FROM metrics_snapshots WHERE timestamp >= %s ORDER BY timestamp ASC`, h.db.Placeholder(1))
	}
	if h.db.Driver == "mssql" {
		q = fmt.Sprintf(`SELECT [timestamp], http_requests, login_success, login_failure, rate_limit_rejections
FROM metrics_snapshots WHERE [timestamp] >= %s ORDER BY [timestamp] ASC`, h.db.Placeholder(1))
	}

	rows, err := h.db.QueryContext(ctx, q, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Snapshot
	for rows.Next() {
		var s Snapshot
		if err := rows.Scan(&s.Timestamp, &s.HTTPRequests, &s.LoginSuccess, &s.LoginFailure, &s.RateLimitRejections); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
