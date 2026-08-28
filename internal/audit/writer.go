// Package audit provides an append-only, async-batched audit log writer.
// Callers push Entry values onto an in-process channel; a background loop
// drains it in batches and inserts them in one multi-row statement so a
// write spike never blocks the request path.
package audit

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/raven-clown/idpforge/internal/db"
)

type Writer struct {
	db            *db.DB
	queue         chan Entry
	batchSize     int
	flushInterval time.Duration
	logger        *slog.Logger

	wg   sync.WaitGroup
	stop chan struct{}
}

func NewWriter(database *db.DB, queueSize, batchSize int, flushInterval time.Duration, logger *slog.Logger) *Writer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Writer{
		db:            database,
		queue:         make(chan Entry, queueSize),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		logger:        logger,
		stop:          make(chan struct{}),
	}
}

// Log enqueues an entry without blocking on the DB. If the queue is full the
// entry is dropped and logged loudly, rather than blocking the caller's
// request — audit logging must never become an outage vector.
func (w *Writer) Log(e Entry) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	select {
	case w.queue <- e:
	default:
		w.logger.Error("audit queue full, dropping entry", "action", e.Action, "actor_id", e.ActorID)
	}
}

// Run drains the queue until ctx is cancelled, then flushes whatever remains.
func (w *Writer) Run(ctx context.Context) {
	w.wg.Add(1)
	defer w.wg.Done()

	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	batch := make([]Entry, 0, w.batchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := w.insertBatch(context.Background(), batch); err != nil {
			w.logger.Error("audit batch insert failed", "count", len(batch), "error", err)
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-w.stop:
			flush()
			return
		case e := <-w.queue:
			batch = append(batch, e)
			if len(batch) >= w.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// Stop signals Run to flush and exit, and blocks until it has.
func (w *Writer) Stop() {
	close(w.stop)
	w.wg.Wait()
}

func (w *Writer) insertBatch(ctx context.Context, entries []Entry) error {
	const cols = 11
	valuesSQL := make([]string, 0, len(entries))
	args := make([]interface{}, 0, len(entries)*cols)

	for i, e := range entries {
		base := i * cols
		ph := make([]string, cols)
		for j := 0; j < cols; j++ {
			ph[j] = w.db.Placeholder(base + j + 1)
		}
		valuesSQL = append(valuesSQL, "("+strings.Join(ph, ",")+")")

		args = append(args,
			nullableString(e.ActorID),
			nullableString(e.ActorIP),
			nullableString(e.ActorUserAgent),
			e.Action,
			nullableString(e.TargetResource),
			nullableString(e.TargetApp),
			nullableBytes(e.BeforeState),
			nullableBytes(e.AfterState),
			e.Status,
			nullableString(e.TraceID),
			e.Timestamp,
		)
	}

	query := fmt.Sprintf(`INSERT INTO audit_logs
(actor_id, actor_ip, actor_user_agent, action, target_resource, target_app, before_state, after_state, status, trace_id, "timestamp")
VALUES %s`, strings.Join(valuesSQL, ","))

	_, err := w.db.ExecContext(ctx, query, args...)
	return err
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullableBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}
