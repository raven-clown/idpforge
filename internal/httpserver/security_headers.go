package httpserver

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/helmet"
)

// securityHeaders sets the response headers a browser actually enforces
// (CSP, frame-ancestors, nosniff, referrer policy, ...) -- previously
// there were none at all, meaning nothing stopped this admin console being
// iframed elsewhere, and an XSS bug would have had an unrestricted blast
// radius.
//
// The CSP allows 'unsafe-inline' for scripts specifically because of the
// theme-init script in web/app/layout.tsx: it has to run inline, before
// first paint, to avoid a light/dark flash, and the static export has no
// per-request server to hand out a nonce. Everything else -- no external
// script/style/image/connect origins, no framing by another site -- is
// still enforced.
func securityHeaders(env string) fiber.Handler {
	return helmet.New(helmet.Config{
		XFrameOptions: "DENY",
		HSTSMaxAge: func() int {
			if env == "development" {
				return 0
			}
			return 15552000 // 180 days
		}(),
		ContentSecurityPolicy: "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: blob:; " +
			"font-src 'self' data:; " +
			"connect-src 'self' ws: wss:; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'",
	})
}
