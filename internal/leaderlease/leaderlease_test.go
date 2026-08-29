package leaderlease

import (
	"context"
	"testing"
	"time"

	"github.com/raven-clown/idpforge/internal/testutil"
)

func TestSingleHolderWinsUncontested(t *testing.T) {
	database := testutil.OpenTestDB(t)
	lease := New(database)
	ctx := context.Background()

	ok, err := lease.TryAcquire(ctx, "test-job", time.Minute)
	if err != nil {
		t.Fatalf("TryAcquire: %v", err)
	}
	if !ok {
		t.Fatal("expected uncontested TryAcquire to succeed")
	}

	// Renewal by the same holder should keep succeeding.
	ok, err = lease.TryAcquire(ctx, "test-job", time.Minute)
	if err != nil {
		t.Fatalf("TryAcquire (renew): %v", err)
	}
	if !ok {
		t.Fatal("expected renewal by the current holder to succeed")
	}
}

func TestSecondHolderLosesWhileLeaseLive(t *testing.T) {
	database := testutil.OpenTestDB(t)
	a := New(database)
	b := New(database)
	ctx := context.Background()

	ok, err := a.TryAcquire(ctx, "test-job", time.Minute)
	if err != nil || !ok {
		t.Fatalf("holder A TryAcquire: ok=%v err=%v", ok, err)
	}

	ok, err = b.TryAcquire(ctx, "test-job", time.Minute)
	if err != nil {
		t.Fatalf("holder B TryAcquire: %v", err)
	}
	if ok {
		t.Fatal("expected holder B to lose while holder A's lease is still live")
	}
}

func TestSecondHolderWinsAfterExpiry(t *testing.T) {
	database := testutil.OpenTestDB(t)
	a := New(database)
	b := New(database)
	ctx := context.Background()

	ok, err := a.TryAcquire(ctx, "test-job", -time.Second) // already expired
	if err != nil || !ok {
		t.Fatalf("holder A TryAcquire: ok=%v err=%v", ok, err)
	}

	ok, err = b.TryAcquire(ctx, "test-job", time.Minute)
	if err != nil {
		t.Fatalf("holder B TryAcquire: %v", err)
	}
	if !ok {
		t.Fatal("expected holder B to win after holder A's lease expired")
	}
}
