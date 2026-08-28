package httpserver

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/apiclient"
	"github.com/raven-clown/idpforge/internal/netutil"
)

// authenticateAny accepts a user session cookie or an X-API-Key, so a
// scoped API client token can call the same /api/v1 admin routes a logged
// in user would, restricted to whatever scopes it was granted.
func (s *Server) authenticateAny(c *fiber.Ctx) error {
	if id := c.Cookies(sessionCookie); id != "" {
		if userID, ok, err := s.sessions.get(c.Context(), id); err == nil && ok {
			c.Locals("user_id", userID)
			return c.Next()
		}
	}

	key := c.Get("X-API-Key")
	if key == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}
	client, err := s.apiClients.Authenticate(c.Context(), key)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid API key")
	}
	if !netutil.IPAllowed(c.IP(), client.AllowedIPs) {
		return fiber.NewError(fiber.StatusForbidden, "source IP not allowed for this API client")
	}
	rlKey := fmt.Sprintf("apiclient_rl:%s:%d", client.ID, time.Now().Unix()/int64(client.RateLimitWindowSeconds))
	if count, err := s.cache.Increment(c.Context(), rlKey, time.Duration(client.RateLimitWindowSeconds)*time.Second); err == nil && count > int64(client.RateLimitMax) {
		return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded for this API client")
	}

	c.Locals("api_client", client)
	return c.Next()
}

// requireUserActor rejects a request that authenticated as an API client
// instead of a real user session; authenticateAny must run first.
func requireUserActor(c *fiber.Ctx) error {
	if uid, ok := c.Locals("user_id").(string); !ok || uid == "" {
		return fiber.NewError(fiber.StatusForbidden, "this endpoint requires a user session, not an API client")
	}
	return c.Next()
}

// requireOwnScopes rejects granting a new API client a scope the issuing
// actor doesn't itself hold, so a token can never escalate past its
// creator's own access.
func (s *Server) requireOwnScopes(c *fiber.Ctx, scopes []string) error {
	for _, scope := range scopes {
		resource, action, ok := splitScope(scope)
		if !ok {
			return fiber.NewError(fiber.StatusBadRequest, "invalid scope "+scope+", want resource:action")
		}

		if userID, ok := c.Locals("user_id").(string); ok && userID != "" {
			allowed, err := s.rbac.HasPermission(c.Context(), userID, resource, action)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "permission check failed")
			}
			if !allowed {
				return fiber.NewError(fiber.StatusForbidden, "cannot grant scope you do not hold: "+scope)
			}
			continue
		}

		if client, ok := c.Locals("api_client").(*apiclient.Client); ok {
			if !client.HasScope(resource, action) {
				return fiber.NewError(fiber.StatusForbidden, "cannot grant scope you do not hold: "+scope)
			}
			continue
		}

		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}
	return nil
}

func splitScope(scope string) (resource, action string, ok bool) {
	for i := len(scope) - 1; i >= 0; i-- {
		if scope[i] == ':' {
			return scope[:i], scope[i+1:], true
		}
	}
	return "", "", false
}

// actorID returns a traceable identifier for whoever authenticated the
// request, a real user ID or "apiclient:<id>", so audit entries always
// have a real actor regardless of which auth path was used.
func actorID(c *fiber.Ctx) string {
	if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
		return uid
	}
	if client, ok := c.Locals("api_client").(*apiclient.Client); ok {
		return "apiclient:" + client.ID
	}
	return ""
}
