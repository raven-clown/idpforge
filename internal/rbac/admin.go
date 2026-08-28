package rbac

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raven-clown/idpforge/internal/db"
)

var ErrNotFound = errors.New("not found")

type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type PermissionRecord struct {
	ID       string `json:"id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type Group struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	ParentGroupID *string `json:"parent_group_id,omitempty"`
}

// Admin manages roles, permissions, groups, and the assignments between
// them and users. Every write invalidates the affected RBAC cache entries
// so permission changes take effect immediately, per the "invalidate on
// change" requirement.
type Admin struct {
	db *db.DB
	r  *Resolver
}

func NewAdmin(database *db.DB, resolver *Resolver) *Admin {
	return &Admin{db: database, r: resolver}
}

func (a *Admin) CreateRole(ctx context.Context, name, description string) (*Role, error) {
	id := uuid.NewString()
	q := fmt.Sprintf(`INSERT INTO roles (id, name, description) VALUES (%s, %s, %s)`,
		a.db.Placeholder(1), a.db.Placeholder(2), a.db.Placeholder(3))
	if _, err := a.db.ExecContext(ctx, q, id, name, description); err != nil {
		return nil, err
	}
	return &Role{ID: id, Name: name, Description: description}, nil
}

func (a *Admin) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT id, name, description FROM roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Role
	for rows.Next() {
		var r Role
		var desc sql.NullString
		if err := rows.Scan(&r.ID, &r.Name, &desc); err != nil {
			return nil, err
		}
		r.Description = desc.String
		out = append(out, r)
	}
	return out, rows.Err()
}

func (a *Admin) DeleteRole(ctx context.Context, id string) error {
	q := fmt.Sprintf(`DELETE FROM roles WHERE id = %s`, a.db.Placeholder(1))
	if _, err := a.db.ExecContext(ctx, q, id); err != nil {
		return err
	}
	return a.r.InvalidateAll(ctx)
}

func (a *Admin) CreatePermission(ctx context.Context, resource, action string) (*PermissionRecord, error) {
	id := uuid.NewString()
	q := fmt.Sprintf(`INSERT INTO permissions (id, resource, action) VALUES (%s, %s, %s)`,
		a.db.Placeholder(1), a.db.Placeholder(2), a.db.Placeholder(3))
	if _, err := a.db.ExecContext(ctx, q, id, resource, action); err != nil {
		return nil, err
	}
	return &PermissionRecord{ID: id, Resource: resource, Action: action}, nil
}

func (a *Admin) ListPermissions(ctx context.Context) ([]PermissionRecord, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT id, resource, action FROM permissions ORDER BY resource, action`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PermissionRecord
	for rows.Next() {
		var p PermissionRecord
		if err := rows.Scan(&p.ID, &p.Resource, &p.Action); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GrantPermissionToRole and the assignment methods below invalidate every
// cached resolution rather than a single user, since a role/group change
// can affect many users at once.

func (a *Admin) GrantPermissionToRole(ctx context.Context, roleID, permissionID string) error {
	q := fmt.Sprintf(`INSERT INTO role_permissions (role_id, permission_id) VALUES (%s, %s)`,
		a.db.Placeholder(1), a.db.Placeholder(2))
	if _, err := a.db.ExecContext(ctx, q, roleID, permissionID); err != nil {
		return err
	}
	return a.r.InvalidateAll(ctx)
}

func (a *Admin) RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error {
	q := fmt.Sprintf(`DELETE FROM role_permissions WHERE role_id = %s AND permission_id = %s`,
		a.db.Placeholder(1), a.db.Placeholder(2))
	if _, err := a.db.ExecContext(ctx, q, roleID, permissionID); err != nil {
		return err
	}
	return a.r.InvalidateAll(ctx)
}

func (a *Admin) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	q := fmt.Sprintf(`INSERT INTO user_roles (user_id, role_id) VALUES (%s, %s)`,
		a.db.Placeholder(1), a.db.Placeholder(2))
	if _, err := a.db.ExecContext(ctx, q, userID, roleID); err != nil {
		return err
	}
	return a.r.Invalidate(ctx, userID)
}

func (a *Admin) RemoveRoleFromUser(ctx context.Context, userID, roleID string) error {
	q := fmt.Sprintf(`DELETE FROM user_roles WHERE user_id = %s AND role_id = %s`,
		a.db.Placeholder(1), a.db.Placeholder(2))
	if _, err := a.db.ExecContext(ctx, q, userID, roleID); err != nil {
		return err
	}
	return a.r.Invalidate(ctx, userID)
}

func (a *Admin) AssignRoleToGroup(ctx context.Context, groupID, roleID string) error {
	q := fmt.Sprintf(`INSERT INTO group_roles (group_id, role_id) VALUES (%s, %s)`,
		a.db.Placeholder(1), a.db.Placeholder(2))
	if _, err := a.db.ExecContext(ctx, q, groupID, roleID); err != nil {
		return err
	}
	return a.r.InvalidateAll(ctx)
}

func (a *Admin) RemoveRoleFromGroup(ctx context.Context, groupID, roleID string) error {
	q := fmt.Sprintf(`DELETE FROM group_roles WHERE group_id = %s AND role_id = %s`,
		a.db.Placeholder(1), a.db.Placeholder(2))
	if _, err := a.db.ExecContext(ctx, q, groupID, roleID); err != nil {
		return err
	}
	return a.r.InvalidateAll(ctx)
}

// CreateGroup models an org unit / "position" that roles and users attach
// to; ParentGroupID gives it a place in the hierarchy the resolver already
// walks (department -> division -> company-wide roles, etc.).
func (a *Admin) CreateGroup(ctx context.Context, name string, parentGroupID *string) (*Group, error) {
	id := uuid.NewString()
	q := fmt.Sprintf(`INSERT INTO groups (id, name, parent_group_id) VALUES (%s, %s, %s)`,
		a.db.Placeholder(1), a.db.Placeholder(2), a.db.Placeholder(3))
	if _, err := a.db.ExecContext(ctx, q, id, name, parentGroupID); err != nil {
		return nil, err
	}
	return &Group{ID: id, Name: name, ParentGroupID: parentGroupID}, nil
}

func (a *Admin) ListGroups(ctx context.Context) ([]Group, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT id, name, parent_group_id FROM groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Group
	for rows.Next() {
		var g Group
		var parent sql.NullString
		if err := rows.Scan(&g.ID, &g.Name, &parent); err != nil {
			return nil, err
		}
		if parent.Valid {
			g.ParentGroupID = &parent.String
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (a *Admin) AddUserToGroup(ctx context.Context, userID, groupID string) error {
	q := fmt.Sprintf(`INSERT INTO user_groups (user_id, group_id) VALUES (%s, %s)`,
		a.db.Placeholder(1), a.db.Placeholder(2))
	if _, err := a.db.ExecContext(ctx, q, userID, groupID); err != nil {
		return err
	}
	return a.r.Invalidate(ctx, userID)
}

func (a *Admin) RemoveUserFromGroup(ctx context.Context, userID, groupID string) error {
	q := fmt.Sprintf(`DELETE FROM user_groups WHERE user_id = %s AND group_id = %s`,
		a.db.Placeholder(1), a.db.Placeholder(2))
	if _, err := a.db.ExecContext(ctx, q, userID, groupID); err != nil {
		return err
	}
	return a.r.Invalidate(ctx, userID)
}
