package httpserver

import (
	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/apiclient"
)

// requirePermission gates a route on the caller's granted permissions:
// a user's resolved RBAC permissions (user -> group(s) -> role(s) ->
// permission(s)), or an API client's explicitly granted scopes when the
// request authenticated via X-API-Key instead of a session.
func (s *Server) requirePermission(resource, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if userID, ok := c.Locals("user_id").(string); ok && userID != "" {
			allowed, err := s.rbac.HasPermission(c.Context(), userID, resource, action)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "permission check failed")
			}
			if !allowed {
				return fiber.NewError(fiber.StatusForbidden, "missing permission "+resource+":"+action)
			}
			return c.Next()
		}

		if client, ok := c.Locals("api_client").(*apiclient.Client); ok {
			if !client.HasScope(resource, action) {
				return fiber.NewError(fiber.StatusForbidden, "API client missing scope "+resource+":"+action)
			}
			return c.Next()
		}

		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}
}
