// Package config loads IdpForge configuration from environment variables
// and resolves OS-appropriate default paths.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type DBDriver string

const (
	DBPostgres DBDriver = "postgres"
	DBMySQL    DBDriver = "mysql"
	DBMSSQL    DBDriver = "mssql"
	DBSQLite   DBDriver = "sqlite"
)

type Config struct {
	Env       string
	HTTP      HTTPConfig
	DB        DBConfig
	Redis     RedisConfig
	Audit     AuditConfig
	Captcha   CaptchaConfig
	OIDC      OIDCConfig
	Backup    BackupConfig
	Paths     PathsConfig
	RateLimit RateLimitConfig
	Storage   StorageConfig
	Bootstrap BootstrapConfig
	// PasswordExpiryDays forces a password change after this many days
	// since it was last set; 0 disables the policy. Applies to idpforge's
	// own SSO login only, not OS-level (Windows/Mac) accounts, which are a
	// separate identity source unless federated via LDAP/AD.
	PasswordExpiryDays int
	// DefaultPassword, if set, is the password every admin-created account
	// or admin-initiated reset gets, always paired with a forced change on
	// next login. It is a server-side config value only: no API accepts,
	// returns, or lets a caller choose an arbitrary password on someone
	// else's behalf. Admins can view it from server config or the settings
	// page (both are "back office" surfaces); it is never exposed through
	// any endpoint that looks up another user's password.
	DefaultPassword string
	// PasswordPolicy applies to any self-service password a user chooses
	// (change-password, forgot-password). It does not gate DefaultPassword,
	// which is an operator-supplied config value, not user input.
	PasswordPolicy PasswordPolicyConfig
	// Timezone is IdpForge's own reference time zone (IANA name, e.g.
	// "Asia/Bangkok"), shown in Settings for schedules/logs. It does not
	// force how timestamps render elsewhere in the UI -- each viewer's
	// browser already renders them in that viewer's own local time, which
	// is what actually handles staff spread across time zones.
	Timezone       string
	UpdateCheck    UpdateCheckConfig
	AccountLockout AccountLockoutConfig
}

// AccountLockoutConfig locks a specific username out of login after too
// many failed attempts within Window, for Duration -- independent of and
// in addition to the existing per-IP rate limit (RateLimitConfig), which
// alone does nothing against an attacker spreading attempts across many
// source IPs.
type AccountLockoutConfig struct {
	MaxAttempts int
	Window      time.Duration
	Duration    time.Duration
}

// UpdateCheckConfig controls the background check against GitHub Releases
// that answers "is a newer IdpForge version out" for self-hosted
// operators, surfaced as an in-app system announcement.
type UpdateCheckConfig struct {
	Enabled  bool
	Interval time.Duration
}

// PasswordPolicyConfig sets the complexity rules enforced on any
// self-chosen password.
type PasswordPolicyConfig struct {
	MinLength        int
	RequireUppercase bool
	RequireLowercase bool
	RequireNumber    bool
	RequireSpecial   bool
}

// BootstrapConfig creates the first admin account on a fresh install (empty
// users table), since every user-creation API route requires an existing
// session. Leave the password unset to have one generated and logged once.
type BootstrapConfig struct {
	AdminUsername string
	AdminEmail    string
	AdminPassword string
}

type HTTPConfig struct {
	ListenAddr string
	BaseURL    string
}

// DBConfig is deliberately driver-agnostic: the same fields describe a
// self-hosted Postgres box, a managed Supabase project (Postgres wire
// protocol), a MySQL/MariaDB server, MSSQL, or a local SQLite file. Point
// DSN at whatever external database the operator already runs.
type DBConfig struct {
	Driver          DBDriver
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	// Enabled=false falls back to the in-memory cache (single-node only).
	Enabled bool
}

type AuditConfig struct {
	BatchSize     int
	FlushInterval time.Duration
	QueueSize     int
}

type CaptchaConfig struct {
	Provider  string // "none", "turnstile", "hcaptcha"
	SiteKey   string
	SecretKey string
}

type OIDCConfig struct {
	Issuer          string
	SigningKeyPath  string
	AccessTokenTTL  time.Duration
	IDTokenTTL      time.Duration
	RefreshTokenTTL time.Duration
}

type BackupConfig struct {
	Enabled   bool
	Dir       string
	Schedule  string // cron expression, consumed by external scheduler (scripts/backup.sh via cron/Task Scheduler)
	Retention int    // days
}

type PathsConfig struct {
	ConfigDir string
	DataDir   string
	LogDir    string
}

// RateLimitConfig covers per-IP request limiting at the app layer. This is
// not DDoS protection: volumetric attacks need to be absorbed upstream (a
// CDN or the cloud provider's network-layer DDoS mitigation) before traffic
// reaches this process. What this does defend against is credential
// stuffing and API abuse from individual clients.
type RateLimitConfig struct {
	Enabled     bool
	Max         int
	Window      time.Duration
	LoginMax    int
	LoginWindow time.Duration
}

// StorageConfig controls where user-uploaded files (profile pictures) land:
// local disk, or an S3-compatible bucket (MinIO, S3, R2, ...).
type StorageConfig struct {
	Backend         string // "local" or "s3"
	LocalDir        string
	S3Endpoint      string
	S3Bucket        string
	S3AccessKey     string
	S3SecretKey     string
	S3UseSSL        bool
	S3PublicBaseURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		Env: getEnv("IDPFORGE_ENV", "development"),
		HTTP: HTTPConfig{
			ListenAddr: getEnv("IDPFORGE_LISTEN_ADDR", ":8080"),
			BaseURL:    getEnv("IDPFORGE_BASE_URL", "http://localhost:8080"),
		},
		DB: DBConfig{
			Driver:          DBDriver(getEnv("IDPFORGE_DB_DRIVER", "postgres")),
			DSN:             getEnv("IDPFORGE_DB_DSN", ""),
			MaxOpenConns:    getEnvInt("IDPFORGE_DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("IDPFORGE_DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("IDPFORGE_DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		Redis: RedisConfig{
			Addr:     getEnv("IDPFORGE_REDIS_ADDR", "localhost:6379"),
			Password: getEnv("IDPFORGE_REDIS_PASSWORD", ""),
			DB:       getEnvInt("IDPFORGE_REDIS_DB", 0),
			Enabled:  getEnvBool("IDPFORGE_REDIS_ENABLED", true),
		},
		Audit: AuditConfig{
			BatchSize:     getEnvInt("IDPFORGE_AUDIT_BATCH_SIZE", 100),
			FlushInterval: getEnvDuration("IDPFORGE_AUDIT_FLUSH_INTERVAL", 2*time.Second),
			QueueSize:     getEnvInt("IDPFORGE_AUDIT_QUEUE_SIZE", 10000),
		},
		Captcha: CaptchaConfig{
			Provider:  getEnv("IDPFORGE_CAPTCHA_PROVIDER", "none"),
			SiteKey:   getEnv("IDPFORGE_CAPTCHA_SITE_KEY", ""),
			SecretKey: getEnv("IDPFORGE_CAPTCHA_SECRET_KEY", ""),
		},
		OIDC: OIDCConfig{
			Issuer:          getEnv("IDPFORGE_OIDC_ISSUER", "http://localhost:8080"),
			SigningKeyPath:  getEnv("IDPFORGE_OIDC_SIGNING_KEY", defaultSigningKeyPath()),
			AccessTokenTTL:  getEnvDuration("IDPFORGE_OIDC_ACCESS_TTL", 15*time.Minute),
			IDTokenTTL:      getEnvDuration("IDPFORGE_OIDC_ID_TTL", 15*time.Minute),
			RefreshTokenTTL: getEnvDuration("IDPFORGE_OIDC_REFRESH_TTL", 30*24*time.Hour),
		},
		Backup: BackupConfig{
			Enabled:   getEnvBool("IDPFORGE_BACKUP_ENABLED", false),
			Dir:       getEnv("IDPFORGE_BACKUP_DIR", defaultBackupDir()),
			Schedule:  getEnv("IDPFORGE_BACKUP_SCHEDULE", "0 3 * * *"),
			Retention: getEnvInt("IDPFORGE_BACKUP_RETENTION_DAYS", 14),
		},
		Paths: PathsConfig{
			ConfigDir: getEnv("IDPFORGE_CONFIG_DIR", defaultConfigDir()),
			DataDir:   getEnv("IDPFORGE_DATA_DIR", defaultDataDir()),
			LogDir:    getEnv("IDPFORGE_LOG_DIR", defaultLogDir()),
		},
		RateLimit: RateLimitConfig{
			Enabled:     getEnvBool("IDPFORGE_RATELIMIT_ENABLED", true),
			Max:         getEnvInt("IDPFORGE_RATELIMIT_MAX", 300),
			Window:      getEnvDuration("IDPFORGE_RATELIMIT_WINDOW", time.Minute),
			LoginMax:    getEnvInt("IDPFORGE_RATELIMIT_LOGIN_MAX", 10),
			LoginWindow: getEnvDuration("IDPFORGE_RATELIMIT_LOGIN_WINDOW", time.Minute),
		},
		Storage: StorageConfig{
			Backend:         getEnv("IDPFORGE_STORAGE_BACKEND", "local"),
			LocalDir:        getEnv("IDPFORGE_STORAGE_LOCAL_DIR", defaultAvatarDir()),
			S3Endpoint:      getEnv("IDPFORGE_STORAGE_S3_ENDPOINT", ""),
			S3Bucket:        getEnv("IDPFORGE_STORAGE_S3_BUCKET", "idpforge-avatars"),
			S3AccessKey:     getEnv("IDPFORGE_STORAGE_S3_ACCESS_KEY", ""),
			S3SecretKey:     getEnv("IDPFORGE_STORAGE_S3_SECRET_KEY", ""),
			S3UseSSL:        getEnvBool("IDPFORGE_STORAGE_S3_USE_SSL", true),
			S3PublicBaseURL: getEnv("IDPFORGE_STORAGE_S3_PUBLIC_BASE_URL", ""),
		},
		Bootstrap: BootstrapConfig{
			AdminUsername: getEnv("IDPFORGE_BOOTSTRAP_ADMIN_USERNAME", "admin"),
			AdminEmail:    getEnv("IDPFORGE_BOOTSTRAP_ADMIN_EMAIL", "admin@localhost"),
			AdminPassword: getEnv("IDPFORGE_BOOTSTRAP_ADMIN_PASSWORD", ""),
		},
		PasswordExpiryDays: getEnvInt("IDPFORGE_PASSWORD_EXPIRY_DAYS", 0),
		DefaultPassword:    getEnv("IDPFORGE_DEFAULT_PASSWORD", ""),
		PasswordPolicy: PasswordPolicyConfig{
			MinLength:        getEnvInt("IDPFORGE_PASSWORD_MIN_LENGTH", 8),
			RequireUppercase: getEnvBool("IDPFORGE_PASSWORD_REQUIRE_UPPERCASE", true),
			RequireLowercase: getEnvBool("IDPFORGE_PASSWORD_REQUIRE_LOWERCASE", true),
			RequireNumber:    getEnvBool("IDPFORGE_PASSWORD_REQUIRE_NUMBER", true),
			RequireSpecial:   getEnvBool("IDPFORGE_PASSWORD_REQUIRE_SPECIAL", true),
		},
		Timezone: getEnv("IDPFORGE_TIMEZONE", "UTC"),
		UpdateCheck: UpdateCheckConfig{
			Enabled:  getEnvBool("IDPFORGE_UPDATE_CHECK_ENABLED", true),
			Interval: getEnvDuration("IDPFORGE_UPDATE_CHECK_INTERVAL", 24*time.Hour),
		},
		AccountLockout: AccountLockoutConfig{
			MaxAttempts: getEnvInt("IDPFORGE_ACCOUNT_LOCKOUT_MAX_ATTEMPTS", 5),
			Window:      getEnvDuration("IDPFORGE_ACCOUNT_LOCKOUT_WINDOW", 15*time.Minute),
			Duration:    getEnvDuration("IDPFORGE_ACCOUNT_LOCKOUT_DURATION", 15*time.Minute),
		},
	}

	if cfg.Storage.Backend != "local" && cfg.Storage.Backend != "s3" {
		return nil, fmt.Errorf("unsupported IDPFORGE_STORAGE_BACKEND %q (want local|s3)", cfg.Storage.Backend)
	}

	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return nil, fmt.Errorf("invalid IDPFORGE_TIMEZONE %q: %w", cfg.Timezone, err)
	}

	if cfg.DB.DSN == "" {
		return nil, fmt.Errorf("IDPFORGE_DB_DSN is required")
	}
	switch cfg.DB.Driver {
	case DBPostgres, DBMySQL, DBMSSQL, DBSQLite:
	default:
		return nil, fmt.Errorf("unsupported IDPFORGE_DB_DRIVER %q (want postgres|mysql|mssql|sqlite)", cfg.DB.Driver)
	}

	return cfg, nil
}

// defaultConfigDir returns %ProgramData%\idpforge on Windows and
// $XDG_CONFIG_HOME/idpforge (or /etc/idpforge) on Linux/Unix.
func defaultConfigDir() string {
	if runtime.GOOS == "windows" {
		base := os.Getenv("ProgramData")
		if base == "" {
			base = `C:\ProgramData`
		}
		return filepath.Join(base, "idpforge")
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "idpforge")
	}
	return filepath.Join("/etc", "idpforge")
}

func defaultDataDir() string {
	if runtime.GOOS == "windows" {
		base := os.Getenv("ProgramData")
		if base == "" {
			base = `C:\ProgramData`
		}
		return filepath.Join(base, "idpforge", "data")
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "idpforge")
	}
	return filepath.Join("/var", "lib", "idpforge")
}

func defaultLogDir() string {
	if runtime.GOOS == "windows" {
		base := os.Getenv("ProgramData")
		if base == "" {
			base = `C:\ProgramData`
		}
		return filepath.Join(base, "idpforge", "logs")
	}
	return filepath.Join("/var", "log", "idpforge")
}

func defaultBackupDir() string {
	return filepath.Join(defaultDataDir(), "backups")
}

func defaultAvatarDir() string {
	return filepath.Join(defaultDataDir(), "avatars")
}

func defaultSigningKeyPath() string {
	return filepath.Join(defaultConfigDir(), "oidc-signing-key.pem")
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

// MaskedDSN returns the DSN with any password component redacted, safe for logging.
func (c DBConfig) MaskedDSN() string {
	dsn := c.DSN
	if i := strings.Index(dsn, "://"); i != -1 {
		rest := dsn[i+3:]
		if at := strings.Index(rest, "@"); at != -1 {
			cred := rest[:at]
			if colon := strings.Index(cred, ":"); colon != -1 {
				return dsn[:i+3] + cred[:colon] + ":***@" + rest[at+1:]
			}
		}
	}
	return dsn
}
