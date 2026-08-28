package httpserver

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/netutil"
)

// requireAPIClient authenticates external integrations (internal web apps,
// other services, AI/automation tools) via X-API-Key, then enforces that
// client's own rate limit, separate from the global per-IP/per-user one.
func (s *Server) requireAPIClient(c *fiber.Ctx) error {
	key := c.Get("X-API-Key")
	if key == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing X-API-Key header")
	}
	client, err := s.apiClients.Authenticate(c.Context(), key)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid API key")
	}
	if !netutil.IPAllowed(c.IP(), client.AllowedIPs) {
		return fiber.NewError(fiber.StatusForbidden, "source IP not allowed for this API client")
	}

	rlKey := fmt.Sprintf("apiclient_rl:%s:%d", client.ID, time.Now().Unix()/int64(client.RateLimitWindowSeconds))
	count, err := s.cache.Increment(c.Context(), rlKey, time.Duration(client.RateLimitWindowSeconds)*time.Second)
	if err == nil && count > int64(client.RateLimitMax) {
		return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded for this API client")
	}

	c.Locals("api_client", client)
	return c.Next()
}
