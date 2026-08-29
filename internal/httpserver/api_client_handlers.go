package httpserver

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/apiclient"
	"github.com/raven-clown/idpforge/internal/audit"
	"github.com/raven-clown/idpforge/internal/users"
)

type createAPIClientRequest struct {
	Name string `json:"name"`
	// Folder is an optional organizational label, purely for grouping in
	// the admin UI's list -- it grants nothing on its own.
	Folder string `json:"folder"`
	// AllowedFields governs the simple /external/v1 read-only path.
	AllowedFields []string `json:"allowed_fields"`
	// Scopes are "resource:action" grants for the full /api/v1 admin API,
	// e.g. ["users:read","users:manage"]. The caller can only grant a scope
	// they themselves hold, so a token can never come back with more power
	// than the admin who issued it.
	Scopes                 []string `json:"scopes"`
	AllowedIPs             []string `json:"allowed_ips"`
	RateLimitMax           int      `json:"rate_limit_max"`
	RateLimitWindowSeconds int      `json:"rate_limit_window_seconds"`
}

func (s *Server) handleCreateAPIClient(c *fiber.Ctx) error {
	var req createAPIClientRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	if req.RateLimitMax <= 0 {
		req.RateLimitMax = 60
	}
	if req.RateLimitWindowSeconds <= 0 {
		req.RateLimitWindowSeconds = 60
	}

	if err := s.requireOwnScopes(c, req.Scopes); err != nil {
		return err
	}

	client, apiKey, err := s.apiClients.Create(c.Context(), req.Name, req.Folder, req.AllowedFields, req.Scopes, req.AllowedIPs, req.RateLimitMax, req.RateLimitWindowSeconds)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "could not create API client")
	}

	s.audit.Log(audit.Entry{
		ActorID:        actorID(c),
		ActorIP:        c.IP(),
		ActorUserAgent: c.Get("User-Agent"),
		Action:         "api_client.create",
		TargetResource: client.ID,
		Status:         "success",
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"client":  client,
		"api_key": apiKey, // shown once
	})
}

func (s *Server) handleListAPIClients(c *fiber.Ctx) error {
	clients, err := s.apiClients.List(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not list API clients")
	}
	return c.JSON(fiber.Map{"clients": clients})
}

func (s *Server) handleDeleteAPIClient(c *fiber.Ctx) error {
	if err := s.apiClients.Delete(c.Context(), c.Params("id")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not delete API client")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// handleExternalLogin lets an integration (an internal web app, or an
// external/AI service) verify a login without implementing OIDC: it
// authenticates via requireAPIClient (X-API-Key + that client's own rate
// limit), checks the username/password, and returns only the fields that
// client is allowed to see.
func (s *Server) handleExternalLogin(c *fiber.Ctx) error {
	client := c.Locals("api_client").(*apiclient.Client)

	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	hash, err := s.users.PasswordHash(c.Context(), req.Username)
	if err != nil || hash == "" || !s.users.VerifyPassword(hash, req.Password) {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}

	user, err := s.users.GetByUsername(c.Context(), req.Username)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}

	return c.JSON(filterUserForClient(user, client.AllowedFields))
}

type externalCreateUserRequest struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	EmployeeID string `json:"employee_id"`
}

// handleExternalCreateUser lets an integration create an account through
// the same simple path it already uses for login/lookup, for apps that
// provision users (an HR system, an onboarding tool) without needing the
// full /api/v1 admin API. Gated behind the users:manage scope, same as
// the admin path -- an /external/v1 client with only read access (no
// scopes) still can't write. Like every other creation path, it never
// accepts a password: the account gets the server-configured default and
// is forced to change it on first login, and the response is filtered
// through the client's own allowed_fields like every other /external/v1
// response.
func (s *Server) handleExternalCreateUser(c *fiber.Ctx) error {
	client := c.Locals("api_client").(*apiclient.Client)
	if !client.HasScope("users", "manage") {
		return fiber.NewError(fiber.StatusForbidden, "this API client is not scoped for users:manage")
	}

	var req externalCreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Username == "" || req.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "username and email are required")
	}
	if s.cfg.DefaultPassword == "" {
		return fiber.NewError(fiber.StatusServiceUnavailable, "IDPFORGE_DEFAULT_PASSWORD is not configured")
	}

	user, err := s.users.Create(c.Context(), users.CreateInput{
		Username:   req.Username,
		Email:      req.Email,
		EmployeeID: req.EmployeeID,
		Password:   s.cfg.DefaultPassword,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "could not create user (username, email, or employee ID already in use?)")
	}

	after, _ := json.Marshal(user)
	s.audit.Log(audit.Entry{
		ActorID:        actorID(c),
		ActorIP:        c.IP(),
		ActorUserAgent: c.Get("User-Agent"),
		Action:         "user.create",
		TargetResource: user.ID,
		AfterState:     after,
		Status:         "success",
	})

	if err := s.rbac.Invalidate(c.Context(), user.ID); err != nil {
		s.log.Warn("rbac cache invalidate failed", "user_id", user.ID, "error", err)
	}

	return c.Status(fiber.StatusCreated).JSON(filterUserForClient(user, client.AllowedFields))
}

func (s *Server) handleExternalGetUser(c *fiber.Ctx) error {
	client := c.Locals("api_client").(*apiclient.Client)

	user, err := s.users.Get(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	return c.JSON(filterUserForClient(user, client.AllowedFields))
}

func filterUserForClient(user interface{}, allowedFields []string) map[string]interface{} {
	encoded, _ := json.Marshal(user)
	var full map[string]interface{}
	_ = json.Unmarshal(encoded, &full)
	return apiclient.FilterFields(full, allowedFields)
}
