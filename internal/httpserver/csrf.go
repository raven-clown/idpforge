package httpserver

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// csrfOriginCheck is a lightweight CSRF defense compatible with the
// SameSite=Lax session cookie: for any state-changing request that
// carries the session cookie, the browser-supplied Origin (falling back
// to Referer, since not every browser sends Origin on same-site
// navigations) must match this server's own origin. A cross-site page
// can trigger a cookie-carrying request, but it cannot control what
// Origin the browser reports for it -- so a mismatch means the request
// didn't originate from this app.
//
// Requests authenticated via X-API-Key or X-Device-Key instead of the
// session cookie are exempt: a page on another site cannot make the
// victim's browser attach a custom header it doesn't know, so those
// aren't CSRF-able the same way.
func csrfOriginCheck(baseURL string) fiber.Handler {
	allowedOrigin := originOf(baseURL)

	return func(c *fiber.Ctx) error {
		if !isStateChanging(c.Method()) {
			return c.Next()
		}
		if c.Cookies(sessionCookie) == "" {
			return c.Next() // not cookie-authenticated, not CSRF-able this way
		}

		origin := c.Get("Origin")
		if origin == "" {
			origin = originOf(c.Get("Referer"))
		}
		if origin == "" || allowedOrigin == "" || origin != allowedOrigin {
			return fiber.NewError(fiber.StatusForbidden, "cross-origin request rejected")
		}
		return c.Next()
	}
}

func isStateChanging(method string) bool {
	switch method {
	case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete:
		return true
	default:
		return false
	}
}

func originOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return strings.ToLower(u.Scheme + "://" + u.Host)
}
