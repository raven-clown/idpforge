package httpserver

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/audit"
	"github.com/raven-clown/idpforge/internal/metrics"
)

const (
	loginFailCachePrefix = "login_fail:"
	loginLockCachePrefix = "login_lock:"
)

// accountLocked reports whether username is currently locked out from
// login attempts, independent of and in addition to the per-IP rate
// limit: that alone does nothing against an attacker spreading attempts
// across many source IPs at one specific account.
func (s *Server) accountLocked(ctx context.Context, username string) bool {
	_, locked, _ := s.cache.Get(ctx, loginLockCachePrefix+username)
	return locked
}

// recordLoginFailure counts a failed attempt against username within the
// configured window, locking the account out once MaxAttempts is reached.
// Disabled entirely when MaxAttempts <= 0.
func (s *Server) recordLoginFailure(ctx context.Context, username string) {
	cfg := s.cfg.AccountLockout
	if cfg.MaxAttempts <= 0 {
		return
	}
	n, err := s.cache.Increment(ctx, loginFailCachePrefix+username, cfg.Window)
	if err != nil || n < int64(cfg.MaxAttempts) {
		return
	}
	_ = s.cache.Set(ctx, loginLockCachePrefix+username, "1", cfg.Duration)
}

func (s *Server) clearLoginFailures(ctx context.Context, username string) {
	_ = s.cache.Delete(ctx, loginFailCachePrefix+username, loginLockCachePrefix+username)
}

// passwordExpired reports whether userID's password is older than the
// configured expiry policy; always false when the policy is disabled (0).
func (s *Server) passwordExpired(c *fiber.Ctx, userID string) bool {
	if s.cfg.PasswordExpiryDays <= 0 {
		return false
	}
	age, err := s.users.PasswordAge(c.Context(), userID)
	if err != nil {
		return false
	}
	return age > time.Duration(s.cfg.PasswordExpiryDays)*24*time.Hour
}

// passwordChangeRequired covers both ways a login can require a change: the
// age-based expiry policy, and the persistent force_password_change flag
// set whenever an account was created or reset to the default password.
func (s *Server) passwordChangeRequired(c *fiber.Ctx, userID string, forced bool) bool {
	return forced || s.passwordExpired(c, userID)
}

type loginRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	MFACode      string `json:"mfa_code"`
	CaptchaToken string `json:"captcha_token"`
}

func (s *Server) handleLogin(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if s.accountLocked(c.Context(), req.Username) {
		s.logFailedLogin(c, req.Username, "locked")
		return fiber.NewError(fiber.StatusTooManyRequests, "too many failed attempts, try again later")
	}

	ok, err := s.captcha.Verify(c.Context(), req.CaptchaToken, c.IP())
	if err != nil || !ok {
		return fiber.NewError(fiber.StatusForbidden, "captcha verification failed")
	}

	hash, err := s.users.PasswordHash(c.Context(), req.Username)
	if err != nil {
		s.logFailedLogin(c, req.Username, "not_found")
		s.recordLoginFailure(c.Context(), req.Username)
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}
	if hash == "" || !s.users.VerifyPassword(hash, req.Password) {
		s.logFailedLogin(c, req.Username, "bad_password")
		s.recordLoginFailure(c.Context(), req.Username)
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}

	user, err := s.users.GetByUsername(c.Context(), req.Username)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}

	if user.MFAEnabled {
		valid, err := s.mfa.Verify(c.Context(), user.ID, req.MFACode)
		if err != nil || !valid {
			s.logFailedLogin(c, req.Username, "mfa_failed")
			s.recordLoginFailure(c.Context(), req.Username)
			return fiber.NewError(fiber.StatusUnauthorized, "invalid MFA code")
		}
	}

	s.clearLoginFailures(c.Context(), req.Username)
	metrics.RecordLoginAttempt("success")

	changeRequired := s.passwordChangeRequired(c, user.ID, user.ForcePasswordChange)
	sessionID, err := s.sessions.create(c.Context(), user.ID, changeRequired)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not create session")
	}
	c.Cookie(&fiber.Cookie{
		Name:     sessionCookie,
		Value:    sessionID,
		HTTPOnly: true,
		Secure:   s.cfg.Env != "development",
		SameSite: "Lax",
		MaxAge:   int(sessionTTL.Seconds()),
	})

	_ = s.users.TouchLastLogin(c.Context(), user.ID)
	s.audit.Log(audit.Entry{
		ActorID:        user.ID,
		ActorIP:        c.IP(),
		ActorUserAgent: c.Get("User-Agent"),
		Action:         "user.login",
		TargetResource: user.ID,
		Status:         "success",
	})

	return c.JSON(fiber.Map{"user_id": user.ID, "mfa_required": user.MFAEnabled, "password_change_required": changeRequired})
}

func (s *Server) logFailedLogin(c *fiber.Ctx, username, reason string) {
	metrics.RecordLoginAttempt(reason)
	after, _ := json.Marshal(fiber.Map{"reason": reason})
	s.audit.Log(audit.Entry{
		ActorIP:        c.IP(),
		ActorUserAgent: c.Get("User-Agent"),
		Action:         "user.login",
		TargetResource: username,
		AfterState:     after,
		Status:         "failure",
	})
}

func (s *Server) handleLogout(c *fiber.Ctx) error {
	id := c.Cookies(sessionCookie)
	if id != "" {
		_ = s.sessions.destroy(c.Context(), id)
	}
	c.ClearCookie(sessionCookie)
	return c.JSON(fiber.Map{"status": "logged_out"})
}
