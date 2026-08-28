// Package httpserver wires the HTTP API: session-based login/MFA/WebAuthn,
// user management CRUD, the OIDC provider endpoints, forward-auth, and
// health/metrics.
package httpserver

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/audit"
	"github.com/raven-clown/idpforge/internal/auth/oidc"
	"github.com/raven-clown/idpforge/internal/cache"
	"github.com/raven-clown/idpforge/internal/captcha"
	"github.com/raven-clown/idpforge/internal/config"
	"github.com/raven-clown/idpforge/internal/mfa"
	"github.com/raven-clown/idpforge/internal/rbac"
	"github.com/raven-clown/idpforge/internal/users"
	"github.com/raven-clown/idpforge/internal/webauthn"
)

type Server struct {
	app *fiber.App

	cfg      *config.Config
	users    *users.Repository
	rbac     *rbac.Resolver
	audit    *audit.Writer
	oidc     *oidc.Provider
	webauthn *webauthn.Service
	mfa      *mfa.Service
	captcha  captcha.Verifier
	sessions *sessionStore
	log      *slog.Logger
}

type Deps struct {
	Config   *config.Config
	Users    *users.Repository
	RBAC     *rbac.Resolver
	Audit    *audit.Writer
	OIDC     *oidc.Provider
	WebAuthn *webauthn.Service
	MFA      *mfa.Service
	Captcha  captcha.Verifier
	Cache    cache.Cache
	Logger   *slog.Logger
}

func New(d Deps) *Server {
	s := &Server{
		cfg:      d.Config,
		users:    d.Users,
		rbac:     d.RBAC,
		audit:    d.Audit,
		oidc:     d.OIDC,
		webauthn: d.WebAuthn,
		mfa:      d.MFA,
		captcha:  d.Captcha,
		sessions: newSessionStore(d.Cache),
		log:      d.Logger,
	}

	s.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          s.errorHandler,
	})

	s.registerRoutes()
	return s
}

func (s *Server) errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if fe, ok := err.(*fiber.Error); ok {
		code = fe.Code
	}
	if code >= 500 {
		s.log.Error("request error", "path", c.Path(), "error", err)
	}
	return c.Status(code).JSON(fiber.Map{"error": err.Error()})
}

func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}

func (s *Server) ShutdownWithTimeout(d time.Duration) error {
	return s.app.ShutdownWithTimeout(d)
}
