// Package leaderlease makes a periodic background job (the update checker,
// the health-alert poller, the metrics sampler) safe to run on every
// instance in a multi-instance deployment without duplicating its work:
// each tick, an instance asks "am I the leader for this job right now?"
// and only the leader actually does the work. Single-instance deployments
// pay a trivial extra DB write per tick and always win their own lease
// uncontested -- there's no separate "cluster mode" flag to turn on.
package leaderlease

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raven-clown/idpforge/internal/db"
)

type Lease struct {
	db       *db.DB
	holderID string
}

// New creates a lease holder identity unique to this process. Call once
// per process and reuse it for every job that needs leader election.
func New(database *db.DB) *Lease {
	return &Lease{db: database, holderID: uuid.NewString()}
}

// TryAcquire reports whether this process is (or just became) the leader
// for jobName, valid until roughly duration from now. Safe to call from
// multiple instances concurrently: exactly one wins each contested claim,
// via the job_name primary key rejecting a second concurrent insert. A
// failed insert's specific error is deliberately never inspected -- the
// follow-up UPDATE is the single source of truth for "do I hold it now,"
// so there's no need to distinguish "lost the race" from any other insert
// failure, and no dialect-specific error-code handling required.
func (l *Lease) TryAcquire(ctx context.Context, jobName string, duration time.Duration) (bool, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(duration)

	del := fmt.Sprintf(`DELETE FROM leader_leases WHERE job_name = %s AND expires_at < %s`,
		l.db.Placeholder(1), l.db.Placeholder(2))
	if _, err := l.db.ExecContext(ctx, del, jobName, now); err != nil {
		return false, err
	}

	ins := fmt.Sprintf(`INSERT INTO leader_leases (job_name, holder_id, expires_at) VALUES (%s, %s, %s)`,
		l.db.Placeholder(1), l.db.Placeholder(2), l.db.Placeholder(3))
	_, _ = l.db.ExecContext(ctx, ins, jobName, l.holderID, expiresAt)

	upd := fmt.Sprintf(`UPDATE leader_leases SET expires_at = %s WHERE job_name = %s AND holder_id = %s`,
		l.db.Placeholder(1), l.db.Placeholder(2), l.db.Placeholder(3))
	res, err := l.db.ExecContext(ctx, upd, expiresAt, jobName, l.holderID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
