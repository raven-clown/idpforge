package db

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// Migrate applies any embedded .sql files for the connected driver that are
// not yet recorded in schema_migrations, in filename order, each inside its
// own transaction. Filenames are expected to sort naturally, e.g.
// 0001_init.sql, 0002_add_mfa.sql.
func (d *DB) Migrate(ctx context.Context) error {
	if err := d.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	migrationsFS, dir, err := migrationsFor(d.Driver)
	if err != nil {
		return err
	}

	entries, err := fs.ReadDir(migrationsFS, dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	applied, err := d.appliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("list applied migrations: %w", err)
	}

	for _, name := range names {
		if applied[name] {
			continue
		}
		content, err := fs.ReadFile(migrationsFS, dir+"/"+name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if err := d.applyMigration(ctx, name, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}

func (d *DB) ensureMigrationsTable(ctx context.Context) error {
	var stmt string
	switch d.Driver {
	case "mssql":
		stmt = `IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='schema_migrations' AND xtype='U')
CREATE TABLE schema_migrations (
	name VARCHAR(255) NOT NULL PRIMARY KEY,
	applied_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
)`
	default:
		stmt = `CREATE TABLE IF NOT EXISTS schema_migrations (
	name VARCHAR(255) NOT NULL PRIMARY KEY,
	applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	}
	_, err := d.ExecContext(ctx, stmt)
	return err
}

func (d *DB) appliedMigrations(ctx context.Context) (map[string]bool, error) {
	rows, err := d.QueryContext(ctx, `SELECT name FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, rows.Err()
}

func (d *DB) applyMigration(ctx context.Context, name, sqlText string) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range splitStatements(sqlText) {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec statement: %w", err)
		}
	}

	insert := `INSERT INTO schema_migrations (name) VALUES (` + d.Placeholder(1) + `)`
	if _, err := tx.ExecContext(ctx, insert, name); err != nil {
		return err
	}
	return tx.Commit()
}

// splitStatements splits a migration file on ";\n" boundaries. Migration
// files must not rely on semicolons inside string literals for stored
// procedures/functions; keep those in driver-specific DO/BEGIN blocks as a
// single statement without an internal ";\n".
func splitStatements(sqlText string) []string {
	return strings.Split(sqlText, ";\n")
}

// Placeholder returns the driver-appropriate bind-parameter marker for
// position n (1-indexed): $1 for Postgres, @p1 for MSSQL, ? for MySQL/SQLite.
func (d *DB) Placeholder(n int) string {
	switch d.Driver {
	case "postgres":
		return fmt.Sprintf("$%d", n)
	case "mssql":
		return fmt.Sprintf("@p%d", n)
	default: // mysql, sqlite
		return "?"
	}
}
