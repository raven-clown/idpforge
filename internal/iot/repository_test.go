package iot

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven-clown/idpforge/internal/testutil"
)

func seedUser(t *testing.T, ctx context.Context, repo *Repository, username string) string {
	t.Helper()
	userID := uuid.NewString()
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO users (id, username, email) VALUES (?, ?, ?)`, userID, username, username+"@example.com"); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return userID
}

func TestDeviceKeyAuthentication(t *testing.T) {
	database := testutil.OpenTestDB(t)
	repo := NewRepository(database)
	ctx := context.Background()

	device, apiKey, err := repo.CreateDevice(ctx, "front-door", "card_reader", "lobby", nil)
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	got, err := repo.AuthenticateDevice(ctx, apiKey)
	if err != nil {
		t.Fatalf("AuthenticateDevice with correct key: %v", err)
	}
	if got.ID != device.ID {
		t.Errorf("AuthenticateDevice returned device %s, want %s", got.ID, device.ID)
	}

	if _, err := repo.AuthenticateDevice(ctx, "wrong-key"); err != ErrNotFound {
		t.Errorf("AuthenticateDevice with wrong key = %v, want ErrNotFound", err)
	}
}

func TestResolveUserSingleFactor(t *testing.T) {
	database := testutil.OpenTestDB(t)
	repo := NewRepository(database)
	ctx := context.Background()

	userID := seedUser(t, ctx, repo, "alice")
	if _, err := repo.AddCredential(ctx, userID, "card", "CARD-001", "badge"); err != nil {
		t.Fatalf("AddCredential: %v", err)
	}

	resolved, err := repo.ResolveUser(ctx, []CredentialProof{{CredentialType: "card", CredentialRef: "CARD-001"}})
	if err != nil {
		t.Fatalf("ResolveUser: %v", err)
	}
	if resolved != userID {
		t.Errorf("ResolveUser = %s, want %s", resolved, userID)
	}
}

func TestResolveUserUnknownCredential(t *testing.T) {
	database := testutil.OpenTestDB(t)
	repo := NewRepository(database)

	_, err := repo.ResolveUser(context.Background(), []CredentialProof{{CredentialType: "face_2d", CredentialRef: "no-such-template"}})
	if err != ErrNotFound {
		t.Errorf("ResolveUser for unknown credential = %v, want ErrNotFound", err)
	}
}

// TestResolveUserMultiFactorMismatch covers the identical-twins scenario:
// a face match and a card swipe that resolve to two different people must
// be rejected, not silently resolved to either one.
func TestResolveUserMultiFactorMismatch(t *testing.T) {
	database := testutil.OpenTestDB(t)
	repo := NewRepository(database)
	ctx := context.Background()

	alice := seedUser(t, ctx, repo, "alice")
	bob := seedUser(t, ctx, repo, "bob")
	repo.AddCredential(ctx, alice, "face_2d", "FACE-ALICE", "")
	repo.AddCredential(ctx, bob, "card", "CARD-BOB", "")

	_, err := repo.ResolveUser(ctx, []CredentialProof{
		{CredentialType: "face_2d", CredentialRef: "FACE-ALICE"},
		{CredentialType: "card", CredentialRef: "CARD-BOB"},
	})
	if err != ErrCredentialMismatch {
		t.Errorf("ResolveUser with mismatched proofs = %v, want ErrCredentialMismatch", err)
	}
}

func TestResolveUserMultiFactorAgreement(t *testing.T) {
	database := testutil.OpenTestDB(t)
	repo := NewRepository(database)
	ctx := context.Background()

	alice := seedUser(t, ctx, repo, "alice")
	repo.AddCredential(ctx, alice, "face_3d", "FACE3D-ALICE", "")
	repo.AddCredential(ctx, alice, "card", "CARD-ALICE", "")

	resolved, err := repo.ResolveUser(ctx, []CredentialProof{
		{CredentialType: "face_3d", CredentialRef: "FACE3D-ALICE"},
		{CredentialType: "card", CredentialRef: "CARD-ALICE"},
	})
	if err != nil {
		t.Fatalf("ResolveUser: %v", err)
	}
	if resolved != alice {
		t.Errorf("ResolveUser = %s, want %s", resolved, alice)
	}
}

func TestHasEventToday(t *testing.T) {
	database := testutil.OpenTestDB(t)
	repo := NewRepository(database)
	ctx := context.Background()

	userID := seedUser(t, ctx, repo, "alice")
	device, _, err := repo.CreateDevice(ctx, "canteen-kiosk", "face_2d", "canteen", nil)
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	before, err := repo.HasEventToday(ctx, userID, "food_discount", "canteen")
	if err != nil {
		t.Fatalf("HasEventToday: %v", err)
	}
	if before {
		t.Fatal("expected no discount event before any check-in")
	}

	if _, err := repo.RecordEvent(ctx, Event{
		DeviceID:  device.ID,
		UserID:    userID,
		EventType: "food_discount",
		Resource:  "canteen",
		Status:    "matched",
		Timestamp: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}

	after, err := repo.HasEventToday(ctx, userID, "food_discount", "canteen")
	if err != nil {
		t.Fatalf("HasEventToday: %v", err)
	}
	if !after {
		t.Error("expected a discount event to be recorded for today after check-in")
	}

	// A different event_type/resource should not be affected.
	unrelated, err := repo.HasEventToday(ctx, userID, "door_access", "canteen")
	if err != nil {
		t.Fatalf("HasEventToday: %v", err)
	}
	if unrelated {
		t.Error("HasEventToday leaked across unrelated event_type")
	}
}
