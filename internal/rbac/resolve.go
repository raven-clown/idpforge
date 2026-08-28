// Package rbac resolves the effective permission set for a user by walking
// user -> group(s) (including ancestor groups) -> role(s) -> permission(s),
// plus any role assigned directly to the user. Resolutions are cached (Redis
// by default, in-memory fallback for single-node setups) and invalidated
// immediately on any RBAC write, per the "cache in Redis, invalidate on
// change" requirement.
package rbac

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/raven-clown/idpforge/internal/cache"
	"github.com/raven-clown/idpforge/internal/db"
)

const (
	cacheKeyPrefix = "rbac:user:"
	defaultTTL     = 5 * time.Minute
)

type Resolver struct {
	db    *db.DB
	cache cache.Cache
	ttl   time.Duration
}

func NewResolver(database *db.DB, c cache.Cache) *Resolver {
	return &Resolver{db: database, cache: c, ttl: defaultTTL}
}

// Resolve returns the effective permission set for userID, serving from
// cache when available.
func (r *Resolver) Resolve(ctx context.Context, userID string) (Resolved, error) {
	key := cacheKeyPrefix + userID
	if cached, ok, err := r.cache.Get(ctx, key); err == nil && ok {
		var resolved Resolved
		if err := json.Unmarshal([]byte(cached), &resolved); err == nil {
			return resolved, nil
		}
	}

	resolved, err := r.resolveFromDB(ctx, userID)
	if err != nil {
		return Resolved{}, err
	}

	if encoded, err := json.Marshal(resolved); err == nil {
		_ = r.cache.Set(ctx, key, string(encoded), r.ttl)
	}
	return resolved, nil
}

// HasPermission is a convenience wrapper around Resolve for the common
// single-check call site (middleware, forward-auth).
func (r *Resolver) HasPermission(ctx context.Context, userID, resource, action string) (bool, error) {
	resolved, err := r.Resolve(ctx, userID)
	if err != nil {
		return false, err
	}
	return resolved.Has(resource, action), nil
}

// Invalidate drops the cached resolution for a single user; call after any
// write that changes that user's own role/group membership.
func (r *Resolver) Invalidate(ctx context.Context, userID string) error {
	return r.cache.Delete(ctx, cacheKeyPrefix+userID)
}

// InvalidateAll drops every cached resolution; call after a write that can
// affect many users at once (role_permissions, group_roles, or a group's
// parent_group_id changing).
func (r *Resolver) InvalidateAll(ctx context.Context) error {
	return r.cache.DeletePrefix(ctx, cacheKeyPrefix)
}

func (r *Resolver) resolveFromDB(ctx context.Context, userID string) (Resolved, error) {
	groupIDs, err := r.userGroupIDs(ctx, userID)
	if err != nil {
		return Resolved{}, fmt.Errorf("load user groups: %w", err)
	}

	allGroupIDs, err := r.expandAncestorGroups(ctx, groupIDs)
	if err != nil {
		return Resolved{}, fmt.Errorf("expand ancestor groups: %w", err)
	}

	roleIDs, roleNames, err := r.roleIDsFor(ctx, userID, allGroupIDs)
	if err != nil {
		return Resolved{}, fmt.Errorf("load roles: %w", err)
	}

	permissions, err := r.permissionsFor(ctx, roleIDs)
	if err != nil {
		return Resolved{}, fmt.Errorf("load permissions: %w", err)
	}

	return Resolved{UserID: userID, RoleNames: roleNames, Permissions: permissions}, nil
}

func (r *Resolver) userGroupIDs(ctx context.Context, userID string) ([]string, error) {
	q := fmt.Sprintf(`SELECT group_id FROM user_groups WHERE user_id = %s`, r.db.Placeholder(1))
	return r.queryIDs(ctx, q, userID)
}

// expandAncestorGroups walks parent_group_id links in Go (not SQL) so the
// same code works identically across Postgres/MySQL/MSSQL/SQLite without a
// dialect-specific recursive CTE.
func (r *Resolver) expandAncestorGroups(ctx context.Context, groupIDs []string) ([]string, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, parent_group_id FROM groups`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	parent := map[string]string{}
	for rows.Next() {
		var id string
		var parentID *string
		if err := rows.Scan(&id, &parentID); err != nil {
			return nil, err
		}
		if parentID != nil {
			parent[id] = *parentID
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var result []string
	for _, gid := range groupIDs {
		cur := gid
		for i := 0; i < 100 && cur != ""; i++ { // depth guard against cyclic data
			if seen[cur] {
				break
			}
			seen[cur] = true
			result = append(result, cur)
			cur = parent[cur]
		}
	}
	return result, nil
}

func (r *Resolver) roleIDsFor(ctx context.Context, userID string, groupIDs []string) ([]string, []string, error) {
	roleIDSet := map[string]bool{}

	direct, err := r.queryIDs(ctx, fmt.Sprintf(`SELECT role_id FROM user_roles WHERE user_id = %s`, r.db.Placeholder(1)), userID)
	if err != nil {
		return nil, nil, err
	}
	for _, id := range direct {
		roleIDSet[id] = true
	}

	if len(groupIDs) > 0 {
		placeholders, args := inClause(r.db, groupIDs, 1)
		q := fmt.Sprintf(`SELECT role_id FROM group_roles WHERE group_id IN (%s)`, placeholders)
		viaGroups, err := r.queryIDs(ctx, q, args...)
		if err != nil {
			return nil, nil, err
		}
		for _, id := range viaGroups {
			roleIDSet[id] = true
		}
	}

	var roleIDs []string
	for id := range roleIDSet {
		roleIDs = append(roleIDs, id)
	}
	if len(roleIDs) == 0 {
		return nil, nil, nil
	}

	placeholders, args := inClause(r.db, roleIDs, 1)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT id, name FROM roles WHERE id IN (%s)`, placeholders), args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, nil, err
		}
		names = append(names, name)
	}
	return roleIDs, names, rows.Err()
}

func (r *Resolver) permissionsFor(ctx context.Context, roleIDs []string) ([]Permission, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}
	placeholders, args := inClause(r.db, roleIDs, 1)
	q := fmt.Sprintf(`SELECT DISTINCT p.resource, p.action
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
WHERE rp.role_id IN (%s)`, placeholders)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.Resource, &p.Action); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *Resolver) queryIDs(ctx context.Context, query string, args ...interface{}) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// inClause builds a driver-appropriate "$1,$2,..." / "?,?,..." placeholder
// list starting at position startAt, plus the matching args slice.
func inClause(database *db.DB, values []string, startAt int) (string, []interface{}) {
	placeholders := make([]string, len(values))
	args := make([]interface{}, len(values))
	for i, v := range values {
		placeholders[i] = database.Placeholder(startAt + i)
		args[i] = v
	}
	return strings.Join(placeholders, ","), args
}
