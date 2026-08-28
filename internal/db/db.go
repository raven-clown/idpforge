// Package db opens a connection to whatever SQL backend the operator points
// IdpForge at: self-hosted Postgres, Supabase or another managed Postgres,
// MySQL/MariaDB, MSSQL, or a local SQLite file for single-node setups. Every
// driver is pure Go, no cgo.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
	_ "modernc.org/sqlite"

	"github.com/raven-clown/idpforge/internal/config"
)

//go:embed migrations/postgres/*.sql
var postgresMigrations embed.FS

//go:embed migrations/mysql/*.sql
var mysqlMigrations embed.FS

//go:embed migrations/mssql/*.sql
var mssqlMigrations embed.FS

//go:embed migrations/sqlite/*.sql
var sqliteMigrations embed.FS

// DB wraps *sql.DB with the resolved driver so callers can branch on
// dialect-specific SQL (e.g. JSONB vs JSON, RETURNING support, upsert syntax).
type DB struct {
	*sql.DB
	Driver config.DBDriver
}

func Open(ctx context.Context, cfg config.DBConfig) (*DB, error) {
	driverName, err := sqlDriverName(cfg.Driver)
	if err != nil {
		return nil, err
	}

	sqlDB, err := sql.Open(driverName, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", cfg.Driver, err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping %s: %w", cfg.Driver, err)
	}

	return &DB{DB: sqlDB, Driver: cfg.Driver}, nil
}

func sqlDriverName(d config.DBDriver) (string, error) {
	switch d {
	case config.DBPostgres:
		return "pgx", nil
	case config.DBMySQL:
		return "mysql", nil
	case config.DBMSSQL:
		return "sqlserver", nil
	case config.DBSQLite:
		return "sqlite", nil
	default:
		return "", fmt.Errorf("unsupported driver %q", d)
	}
}

// migrationsFor returns the embedded migration set matching the driver, since
// dialects diverge on JSON column types, partitioning, and upsert syntax.
func migrationsFor(d config.DBDriver) (embed.FS, string, error) {
	switch d {
	case config.DBPostgres:
		return postgresMigrations, "migrations/postgres", nil
	case config.DBMySQL:
		return mysqlMigrations, "migrations/mysql", nil
	case config.DBMSSQL:
		return mssqlMigrations, "migrations/mssql", nil
	case config.DBSQLite:
		return sqliteMigrations, "migrations/sqlite", nil
	default:
		return embed.FS{}, "", fmt.Errorf("unsupported driver %q", d)
	}
}

var _ = stdlib.GetDefaultDriver // keep pgx stdlib import referenced
