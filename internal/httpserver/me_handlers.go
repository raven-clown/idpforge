package httpserver

import (
	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/users"
)

// handleMe backs the SPA's "who am I" check on load: current user plus
// whether a password change is required before anything else.
func (s *Server) handleMe(c *fiber.Ctx) error {
	sessionID := c.Cookies(sessionCookie)
	data, ok, err := s.sessions.getFull(c.Context(), sessionID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "session lookup failed")
	}
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	user, err := s.users.Get(c.Context(), data.UserID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	// The frontend uses this to decide what to even show: hiding a nav
	// link or an action button the user can't use is better than
	// rendering it and letting the API 403 it after the fact.
	resolved, err := s.rbac.Resolve(c.Context(), data.UserID)
	permissions := []string{}
	if err == nil {
		for _, p := range resolved.Permissions {
			permissions = append(permissions, p.Resource+":"+p.Action)
		}
	}

	return c.JSON(fiber.Map{
		"user":                 user,
		"must_change_password": data.MustChangePassword,
		"version":              s.version,
		"permissions":          permissions,
	})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (s *Server) handleChangePassword(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var req changePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := users.ValidatePassword(req.NewPassword, s.cfg.PasswordPolicy); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	user, err := s.users.Get(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}
	hash, err := s.users.PasswordHash(c.Context(), user.Username)
	if err != nil || hash == "" || !s.users.VerifyPassword(hash, req.CurrentPassword) {
		return fiber.NewError(fiber.StatusUnauthorized, "current password is incorrect")
	}

	if err := s.users.SetPassword(c.Context(), userID, req.NewPassword); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not change password")
	}

	sessionID := c.Cookies(sessionCookie)
	_ = s.sessions.clearMustChangePassword(c.Context(), sessionID, userID)

	return c.JSON(fiber.Map{"status": "changed"})
}
