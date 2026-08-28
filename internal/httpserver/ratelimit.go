package httpserver

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"github.com/raven-clown/idpforge/internal/config"
	"github.com/raven-clown/idpforge/internal/metrics"
)

// rateLimiter keys by authenticated user ID when a session cookie resolves
// to one, falling back to client IP for anonymous traffic (login,
// discovery, JWKS).
func rateLimiter(sessions *sessionStore, max int, window time.Duration, routeLabel string) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: window,
		KeyGenerator: func(c *fiber.Ctx) string {
			if id := c.Cookies(sessionCookie); id != "" {
				if userID, ok, err := sessions.get(c.Context(), id); err == nil && ok {
					return "user:" + userID
				}
			}
			return "ip:" + c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			metrics.RecordRateLimitRejection(routeLabel)
			return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded, try again later")
		},
	})
}

func rateLimitMiddlewares(cfg config.RateLimitConfig, sessions *sessionStore) (global, login fiber.Handler) {
	if !cfg.Enabled {
		noop := func(c *fiber.Ctx) error { return c.Next() }
		return noop, noop
	}
	return rateLimiter(sessions, cfg.Max, cfg.Window, "global"),
		rateLimiter(sessions, cfg.LoginMax, cfg.LoginWindow, "login")
}
