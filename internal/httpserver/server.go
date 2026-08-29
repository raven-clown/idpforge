// Package httpserver wires the HTTP API: session-based login/MFA/WebAuthn,
// user management CRUD, the OIDC provider endpoints, forward-auth,
// health/metrics, and the embedded admin SPA.
package httpserver

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/raven-clown/idpforge/internal/announcements"
	"github.com/raven-clown/idpforge/internal/apiclient"
	"github.com/raven-clown/idpforge/internal/audit"
	"github.com/raven-clown/idpforge/internal/auth/oidc"
	"github.com/raven-clown/idpforge/internal/cache"
	"github.com/raven-clown/idpforge/internal/captcha"
	"github.com/raven-clown/idpforge/internal/config"
	"github.com/raven-clown/idpforge/internal/health"
	"github.com/raven-clown/idpforge/internal/iot"
	"github.com/raven-clown/idpforge/internal/metrics"
	"github.com/raven-clown/idpforge/internal/mfa"
	"github.com/raven-clown/idpforge/internal/rbac"
	"github.com/raven-clown/idpforge/internal/storage"
	"github.com/raven-clown/idpforge/internal/users"
	"github.com/raven-clown/idpforge/internal/webauthn"
)

const requestIDContextKey = "requestid" // matches requestid.ConfigDefault.ContextKey

type Server struct {
	app *fiber.App

	cfg         *config.Config
	users       *users.Repository
	rbac        *rbac.Resolver
	rbacAdm     *rbac.Admin
	audit       *audit.Writer
	auditReader *audit.Reader
	metricsHist *metrics.History
	oidc        *oidc.Provider
	webauthn    *webauthn.Service
	mfa         *mfa.Service
	captcha     captcha.Verifier
	storage     storage.Store
	health      *health.Checker
	iot         *iot.Repository
	apiClients  *apiclient.Repository
	announce    *announcements.Repository
	sessions    *sessionStore
	cache       cache.Cache
	log         *slog.Logger
	hub         *realtimeHub
	version     string

	rateLimitGlobal fiber.Handler
	rateLimitLogin  fiber.Handler
}

type Deps struct {
	Config      *config.Config
	Users       *users.Repository
	RBAC        *rbac.Resolver
	RBACAdm     *rbac.Admin
	Audit       *audit.Writer
	AuditReader *audit.Reader
	MetricsHist *metrics.History
	OIDC        *oidc.Provider
	WebAuthn    *webauthn.Service
	MFA         *mfa.Service
	Captcha     captcha.Verifier
	Storage     storage.Store
	Health      *health.Checker
	IoT         *iot.Repository
	APIClients  *apiclient.Repository
	Announce    *announcements.Repository
	Cache       cache.Cache
	Logger      *slog.Logger
	Version     string
}

func New(d Deps) *Server {
	s := &Server{
		cfg:         d.Config,
		users:       d.Users,
		rbac:        d.RBAC,
		rbacAdm:     d.RBACAdm,
		audit:       d.Audit,
		auditReader: d.AuditReader,
		metricsHist: d.MetricsHist,
		oidc:        d.OIDC,
		webauthn:    d.WebAuthn,
		mfa:         d.MFA,
		captcha:     d.Captcha,
		storage:     d.Storage,
		health:      d.Health,
		iot:         d.IoT,
		apiClients:  d.APIClients,
		announce:    d.Announce,
		sessions:    newSessionStore(d.Cache),
		cache:       d.Cache,
		log:         d.Logger,
		hub:         newRealtimeHub(redisCacheOrNil(d.Cache)),
		version:     d.Version,
	}
	if d.Audit != nil {
		d.Audit.SetOnLog(s.hub.onAuditLog)
	}

	s.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          s.errorHandler,
	})
	s.app.Use(requestid.New())
	s.app.Use(metricsMiddleware)
	s.app.Use(securityHeaders(d.Config.Env))
	s.app.Use(csrfOriginCheck(d.Config.HTTP.BaseURL))

	global, login := rateLimitMiddlewares(d.Config.RateLimit, s.sessions)
	s.rateLimitGlobal = global
	s.rateLimitLogin = login

	s.registerRoutes()
	return s
}

// errorHandler logs the full error server-side (path, method, IP, request
// ID) so the actual cause is findable. Clients only get a generic message
// plus that ID for 5xx, since the underlying error text can contain
// internal details (DSNs, file paths, driver errors).
func (s *Server) errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if fe, ok := err.(*fiber.Error); ok {
		code = fe.Code
	}

	reqID, _ := c.Locals(requestIDContextKey).(string)

	if code >= 500 {
		s.log.Error("request failed",
			"request_id", reqID,
			"method", c.Method(),
			"path", c.Path(),
			"ip", c.IP(),
			"error", err,
		)
		return c.Status(code).JSON(fiber.Map{
			"error":      "internal_error",
			"message":    "something went wrong on our end",
			"request_id": reqID,
		})
	}

	return c.Status(code).JSON(fiber.Map{"error": err.Error(), "request_id": reqID})
}

func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}

func (s *Server) ShutdownWithTimeout(d time.Duration) error {
	return s.app.ShutdownWithTimeout(d)
}

// redisCacheOrNil returns c as a *cache.RedisCache when Redis is actually
// in use, or nil for the in-memory single-node fallback -- realtimeHub
// uses this to decide whether it can fan broadcasts out across instances.
func redisCacheOrNil(c cache.Cache) *cache.RedisCache {
	rc, _ := c.(*cache.RedisCache)
	return rc
}
