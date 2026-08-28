package rbac

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/raven-clown/idpforge/internal/cache"
	"github.com/raven-clown/idpforge/internal/db"
	"github.com/raven-clown/idpforge/internal/testutil"
)

// seedFixture builds: company (parent) -> engineering (child), with user1
// a member of engineering only. company grants role "employee" (intranet:
// read); engineering grants role "developer" (gitlab:admin). user1 also
// gets a direct role "oncall" (pagerduty:ack), exercising the user_roles
// override path independent of group membership.
func seedFixture(t *testing.T, database *db.DB) (userID string) {
	t.Helper()
	ctx := context.Background()
	exec := func(q string, args ...interface{}) {
		t.Helper()
		if _, err := database.ExecContext(ctx, q, args...); err != nil {
			t.Fatalf("seed exec failed (%s): %v", q, err)
		}
	}

	userID = uuid.NewString()
	companyID := uuid.NewString()
	engineeringID := uuid.NewString()
	employeeRoleID := uuid.NewString()
	developerRoleID := uuid.NewString()
	oncallRoleID := uuid.NewString()
	intranetPermID := uuid.NewString()
	gitlabPermID := uuid.NewString()
	pagerdutyPermID := uuid.NewString()

	exec(`INSERT INTO users (id, username, email) VALUES (?, ?, ?)`, userID, "alice", "alice@example.com")
	exec(`INSERT INTO groups (id, name, parent_group_id) VALUES (?, ?, NULL)`, companyID, "company")
	exec(`INSERT INTO groups (id, name, parent_group_id) VALUES (?, ?, ?)`, engineeringID, "engineering", companyID)
	exec(`INSERT INTO user_groups (user_id, group_id) VALUES (?, ?)`, userID, engineeringID)

	exec(`INSERT INTO roles (id, name) VALUES (?, ?)`, employeeRoleID, "employee")
	exec(`INSERT INTO roles (id, name) VALUES (?, ?)`, developerRoleID, "developer")
	exec(`INSERT INTO roles (id, name) VALUES (?, ?)`, oncallRoleID, "oncall")

	exec(`INSERT INTO permissions (id, resource, action) VALUES (?, ?, ?)`, intranetPermID, "intranet", "read")
	exec(`INSERT INTO permissions (id, resource, action) VALUES (?, ?, ?)`, gitlabPermID, "gitlab", "admin")
	exec(`INSERT INTO permissions (id, resource, action) VALUES (?, ?, ?)`, pagerdutyPermID, "pagerduty", "ack")

	exec(`INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)`, employeeRoleID, intranetPermID)
	exec(`INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)`, developerRoleID, gitlabPermID)
	exec(`INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)`, oncallRoleID, pagerdutyPermID)

	exec(`INSERT INTO group_roles (group_id, role_id) VALUES (?, ?)`, companyID, employeeRoleID)
	exec(`INSERT INTO group_roles (group_id, role_id) VALUES (?, ?)`, engineeringID, developerRoleID)
	exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, userID, oncallRoleID)

	return userID
}

func TestResolveWalksGroupHierarchyAndDirectRoles(t *testing.T) {
	database := testutil.OpenTestDB(t)
	userID := seedFixture(t, database)

	resolver := NewResolver(database, cache.NewMemory())
	resolved, err := resolver.Resolve(context.Background(), userID)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	wantPerms := map[string]bool{"intranet:read": false, "gitlab:admin": false, "pagerduty:ack": false}
	for _, p := range resolved.Permissions {
		key := p.String()
		if _, ok := wantPerms[key]; ok {
			wantPerms[key] = true
		}
	}
	for perm, found := range wantPerms {
		if !found {
			t.Errorf("expected resolved permissions to include %s (via group hierarchy or direct role), got %v", perm, resolved.Permissions)
		}
	}
	if len(resolved.Permissions) != 3 {
		t.Errorf("expected exactly 3 resolved permissions, got %d: %v", len(resolved.Permissions), resolved.Permissions)
	}
}

func TestResolveIsCachedAndInvalidateAllForcesRefresh(t *testing.T) {
	database := testutil.OpenTestDB(t)
	userID := seedFixture(t, database)

	c := cache.NewMemory()
	resolver := NewResolver(database, c)
	ctx := context.Background()

	first, err := resolver.Resolve(ctx, userID)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !first.Has("gitlab", "admin") {
		t.Fatal("expected initial resolution to include gitlab:admin")
	}

	// Grant a new permission directly in the DB without touching the
	// resolver; the cached result should NOT change until invalidated.
	newPermID := uuid.NewString()
	if _, err := database.ExecContext(ctx, `INSERT INTO permissions (id, resource, action) VALUES (?, ?, ?)`, newPermID, "vault", "read"); err != nil {
		t.Fatalf("insert permission: %v", err)
	}
	rows, err := database.QueryContext(ctx, `SELECT role_id FROM user_roles WHERE user_id = ?`, userID)
	if err != nil {
		t.Fatalf("query user_roles: %v", err)
	}
	var oncallRoleID string
	if rows.Next() {
		rows.Scan(&oncallRoleID)
	}
	rows.Close()
	if _, err := database.ExecContext(ctx, `INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)`, oncallRoleID, newPermID); err != nil {
		t.Fatalf("insert role_permission: %v", err)
	}

	stale, err := resolver.Resolve(ctx, userID)
	if err != nil {
		t.Fatalf("Resolve (should hit cache): %v", err)
	}
	if stale.Has("vault", "read") {
		t.Error("expected cached resolution to be stale (not reflect the new grant yet)")
	}

	if err := resolver.InvalidateAll(ctx); err != nil {
		t.Fatalf("InvalidateAll: %v", err)
	}

	fresh, err := resolver.Resolve(ctx, userID)
	if err != nil {
		t.Fatalf("Resolve (post-invalidate): %v", err)
	}
	if !fresh.Has("vault", "read") {
		t.Error("expected resolution after InvalidateAll to include the newly granted permission")
	}
}

func TestResolveUnknownUserReturnsNoPermissions(t *testing.T) {
	database := testutil.OpenTestDB(t)
	resolver := NewResolver(database, cache.NewMemory())

	resolved, err := resolver.Resolve(context.Background(), uuid.NewString())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resolved.Permissions) != 0 {
		t.Errorf("expected no permissions for an unknown user, got %v", resolved.Permissions)
	}
}
