// Package bootstrap creates the first admin account on a fresh install.
// Every user-management API route requires an existing session, so
// something has to create account zero outside that loop.
package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/raven-clown/idpforge/internal/config"
	"github.com/raven-clown/idpforge/internal/db"
	"github.com/raven-clown/idpforge/internal/rbac"
	"github.com/raven-clown/idpforge/internal/users"
)

// adminPermissions are granted to the bootstrap "admin" role, matching
// every resource:action pair the built-in HTTP routes check.
var adminPermissions = [][2]string{
	{"users", "read"}, {"users", "manage"},
	{"rbac", "manage"},
	{"iot", "read"}, {"iot", "manage"},
	{"api_clients", "manage"},
}

// Run creates the first admin user, role, and permissions if the users
// table is empty. A no-op on every subsequent start.
func Run(ctx context.Context, database *db.DB, userRepo *users.Repository, rbacAdm *rbac.Admin, cfg config.BootstrapConfig, logger *slog.Logger) error {
	var count int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	password := cfg.AdminPassword
	generated := password == ""
	if generated {
		var err error
		password, err = randomPassword()
		if err != nil {
			return fmt.Errorf("generate bootstrap password: %w", err)
		}
	}

	admin, err := userRepo.Create(ctx, users.CreateInput{
		Username: cfg.AdminUsername,
		Email:    cfg.AdminEmail,
		Password: password,
	})
	if err != nil {
		return fmt.Errorf("create bootstrap admin: %w", err)
	}

	role, err := rbacAdm.CreateRole(ctx, "admin", "Full access, created on first run")
	if err != nil {
		return fmt.Errorf("create bootstrap admin role: %w", err)
	}

	for _, pa := range adminPermissions {
		perm, err := rbacAdm.CreatePermission(ctx, pa[0], pa[1])
		if err != nil {
			return fmt.Errorf("create permission %s:%s: %w", pa[0], pa[1], err)
		}
		if err := rbacAdm.GrantPermissionToRole(ctx, role.ID, perm.ID); err != nil {
			return fmt.Errorf("grant %s:%s to admin role: %w", pa[0], pa[1], err)
		}
	}

	if err := rbacAdm.AssignRoleToUser(ctx, admin.ID, role.ID); err != nil {
		return fmt.Errorf("assign admin role: %w", err)
	}

	if generated {
		logger.Warn("bootstrap admin account created with a generated password, save it now",
			"username", cfg.AdminUsername, "password", password)
	} else {
		logger.Info("bootstrap admin account created", "username", cfg.AdminUsername)
	}
	return nil
}

func randomPassword() (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
