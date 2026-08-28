package httpserver

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/audit"
	"github.com/raven-clown/idpforge/internal/metrics"
)

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

	ok, err := s.captcha.Verify(c.Context(), req.CaptchaToken, c.IP())
	if err != nil || !ok {
		return fiber.NewError(fiber.StatusForbidden, "captcha verification failed")
	}

	hash, err := s.users.PasswordHash(c.Context(), req.Username)
	if err != nil {
		s.logFailedLogin(c, req.Username, "not_found")
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}
	if hash == "" || !s.users.VerifyPassword(hash, req.Password) {
		s.logFailedLogin(c, req.Username, "bad_password")
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
			return fiber.NewError(fiber.StatusUnauthorized, "invalid MFA code")
		}
	}

	metrics.LoginAttemptsTotal.WithLabelValues("success").Inc()

	expired := s.passwordExpired(c, user.ID)
	sessionID, err := s.sessions.create(c.Context(), user.ID, expired)
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

	return c.JSON(fiber.Map{"user_id": user.ID, "mfa_required": user.MFAEnabled, "password_change_required": expired})
}

func (s *Server) logFailedLogin(c *fiber.Ctx, username, reason string) {
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
