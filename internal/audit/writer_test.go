package audit

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/raven-clown/idpforge/internal/testutil"
)

func TestWriterBatchesAndFlushesOnSize(t *testing.T) {
	database := testutil.OpenTestDB(t)
	w := NewWriter(database, 100, 3, time.Hour, slog.Default()) // long flush interval: only size-based flush should fire

	ctx, cancel := context.WithCancel(context.Background())
	w.Run(ctx)
	t.Cleanup(func() { w.Stop(); cancel() })

	for i := 0; i < 3; i++ {
		w.Log(Entry{Action: "test.action", Status: "success"})
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		var count int
		if err := database.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM audit_logs`).Scan(&count); err != nil {
			t.Fatalf("count audit_logs: %v", err)
		}
		if count == 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected 3 audit rows after batch flush, got %d", count)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestWriterFlushesOnStop(t *testing.T) {
	database := testutil.OpenTestDB(t)
	w := NewWriter(database, 100, 100, time.Hour, slog.Default()) // batch size never reached; must flush on Stop

	ctx, cancel := context.WithCancel(context.Background())
	w.Run(ctx)

	w.Log(Entry{Action: "test.stop_flush", Status: "success"})
	w.Stop()
	cancel()

	var count int
	if err := database.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM audit_logs WHERE action = 'test.stop_flush'`).Scan(&count); err != nil {
		t.Fatalf("count audit_logs: %v", err)
	}
	if count != 1 {
		t.Errorf("expected the queued entry to be flushed on Stop, got %d rows", count)
	}
}

func TestWriterDropsEntriesWhenQueueFull(t *testing.T) {
	database := testutil.OpenTestDB(t)
	w := NewWriter(database, 1, 100, time.Hour, slog.Default()) // queue size 1, no consumer running

	w.Log(Entry{Action: "first", Status: "success"})
	w.Log(Entry{Action: "dropped", Status: "success"}) // queue full, should be dropped without blocking

	// The call above must not have blocked; if we got here, it didn't.
}
