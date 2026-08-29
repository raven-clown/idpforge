package httpserver

import "github.com/gofiber/fiber/v2"

// handleGetSettings returns a read-only, secret-free view of the running
// configuration. Almost all of it is set via environment variables and
// needs a restart to change, same as nginx.conf or postgresql.conf, so
// this deliberately isn't an editable form.
func (s *Server) handleGetSettings(c *fiber.Ctx) error {
	cfg := s.cfg
	return c.JSON(fiber.Map{
		"env":      cfg.Env,
		"version":  s.version,
		"timezone": cfg.Timezone,
		"http": fiber.Map{
			"listen_addr": cfg.HTTP.ListenAddr,
			"base_url":    cfg.HTTP.BaseURL,
		},
		"database": fiber.Map{
			"driver": cfg.DB.Driver,
			"dsn":    cfg.DB.MaskedDSN(),
		},
		"redis": fiber.Map{
			"enabled": cfg.Redis.Enabled,
			"addr":    cfg.Redis.Addr,
		},
		"rate_limit": fiber.Map{
			"enabled":              cfg.RateLimit.Enabled,
			"max":                  cfg.RateLimit.Max,
			"window_seconds":       cfg.RateLimit.Window.Seconds(),
			"login_max":            cfg.RateLimit.LoginMax,
			"login_window_seconds": cfg.RateLimit.LoginWindow.Seconds(),
		},
		"captcha": fiber.Map{
			"provider": cfg.Captcha.Provider,
		},
		"oidc": fiber.Map{
			"issuer":                   cfg.OIDC.Issuer,
			"access_token_ttl_minutes": cfg.OIDC.AccessTokenTTL.Minutes(),
			"id_token_ttl_minutes":     cfg.OIDC.IDTokenTTL.Minutes(),
			"refresh_token_ttl_hours":  cfg.OIDC.RefreshTokenTTL.Hours(),
		},
		"backup": fiber.Map{
			"enabled":        cfg.Backup.Enabled,
			"dir":            cfg.Backup.Dir,
			"schedule":       cfg.Backup.Schedule,
			"retention_days": cfg.Backup.Retention,
		},
		"storage": fiber.Map{
			"backend": cfg.Storage.Backend,
		},
		"password_expiry_days": cfg.PasswordExpiryDays,
		// default_password is intentionally shown here: this endpoint is the
		// admin-only back office view (settings:read), and it is the one
		// place besides server config (IDPFORGE_DEFAULT_PASSWORD) an admin
		// can see it. It is still never exposed for any individual user's
		// actual current password.
		"default_password": cfg.DefaultPassword,
		"password_policy": fiber.Map{
			"min_length":        cfg.PasswordPolicy.MinLength,
			"require_uppercase": cfg.PasswordPolicy.RequireUppercase,
			"require_lowercase": cfg.PasswordPolicy.RequireLowercase,
			"require_number":    cfg.PasswordPolicy.RequireNumber,
			"require_special":   cfg.PasswordPolicy.RequireSpecial,
		},
	})
}
