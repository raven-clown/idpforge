// Package testutil provides a real, migrated SQLite database for tests
// that exercise actual SQL (RBAC resolution, IoT credential matching)
// instead of mocking database/sql.
package testutil

import (
	"context"
	"testing"

	"github.com/raven-clown/idpforge/internal/config"
	"github.com/raven-clown/idpforge/internal/db"
)

// OpenTestDB returns a migrated, in-memory SQLite database scoped to one
// connection (cache=shared + MaxOpenConns=1) so every query in the test
// sees the same in-memory instance.
func OpenTestDB(t *testing.T) *db.DB {
	t.Helper()

	cfg := config.DBConfig{
		Driver:       config.DBSQLite,
		DSN:          "file::memory:?cache=shared",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}

	database, err := db.Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := database.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return database
}
