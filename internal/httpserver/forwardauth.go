package httpserver

import "github.com/gofiber/fiber/v2"

// handleForwardAuth backs Traefik's forwardAuth middleware for legacy apps
// with no native SSO support: on a valid session it returns 200 and injects
// X-Forwarded-User / X-Forwarded-Groups headers for the upstream app; on no
// session it returns 401 so Traefik redirects to /login.
func (s *Server) handleForwardAuth(c *fiber.Ctx) error {
	sessionID := c.Cookies(sessionCookie)
	if sessionID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	userID, ok, err := s.sessions.get(c.Context(), sessionID)
	if err != nil || !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "session expired")
	}

	resolved, err := s.rbac.Resolve(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "permission resolve failed")
	}

	c.Set("X-Forwarded-User", userID)
	for _, role := range resolved.RoleNames {
		c.Response().Header.Add("X-Forwarded-Groups", role)
	}
	return c.SendStatus(fiber.StatusOK)
}
