package httpserver

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/audit"
	"github.com/raven-clown/idpforge/internal/users"
)

type createUserRequest struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	EmployeeID string `json:"employee_id"`
}

// handleCreateUser never accepts a caller-chosen password: every new
// account is created with the server-configured default password
// (IDPFORGE_DEFAULT_PASSWORD) and force_password_change set, so the first
// thing a new employee does is set their own.
func (s *Server) handleCreateUser(c *fiber.Ctx) error {
	var req createUserRequest
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

	return c.Status(fiber.StatusCreated).JSON(user)
}

// handleResetUserPassword resets a user's password back to the
// server-configured default and forces a change on their next login.
// Like create, it takes no password in the request body: there is no API
// path to set an arbitrary password for someone else, and no API path to
// read one back — this handler returns the User object only, never the
// password value.
func (s *Server) handleResetUserPassword(c *fiber.Ctx) error {
	id := c.Params("id")
	if s.cfg.DefaultPassword == "" {
		return fiber.NewError(fiber.StatusServiceUnavailable, "IDPFORGE_DEFAULT_PASSWORD is not configured")
	}

	if err := s.users.AssignDefaultPassword(c.Context(), id, s.cfg.DefaultPassword); err != nil {
		if err == users.ErrNotFound {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "could not reset password")
	}

	user, err := s.users.Get(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	s.audit.Log(audit.Entry{
		ActorID:        actorID(c),
		ActorIP:        c.IP(),
		ActorUserAgent: c.Get("User-Agent"),
		Action:         "user.reset_password",
		TargetResource: id,
		Status:         "success",
	})

	return c.JSON(user)
}

func (s *Server) handleGetUser(c *fiber.Ctx) error {
	user, err := s.users.Get(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}
	return c.JSON(user)
}

func (s *Server) handleListUsers(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 500 {
		limit = 50
	}

	list, err := s.users.List(c.Context(), limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not list users")
	}
	return c.JSON(fiber.Map{"users": list, "limit": limit, "offset": offset})
}

type updateUserRequest struct {
	Email      *string `json:"email"`
	EmployeeID *string `json:"employee_id"`
	Status     *string `json:"status"`
}

func (s *Server) handleUpdateUser(c *fiber.Ctx) error {
	var req updateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	before, _ := s.users.Get(c.Context(), c.Params("id"))

	in := users.UpdateInput{Email: req.Email, EmployeeID: req.EmployeeID}
	if req.Status != nil {
		st := users.Status(*req.Status)
		in.Status = &st
	}

	user, err := s.users.Update(c.Context(), c.Params("id"), in)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(user)
	s.audit.Log(audit.Entry{
		ActorID:        actorID(c),
		ActorIP:        c.IP(),
		ActorUserAgent: c.Get("User-Agent"),
		Action:         "user.update",
		TargetResource: user.ID,
		BeforeState:    beforeJSON,
		AfterState:     afterJSON,
		Status:         "success",
	})

	return c.JSON(user)
}

func (s *Server) handleDeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	before, _ := s.users.Get(c.Context(), id)

	if err := s.users.Delete(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	beforeJSON, _ := json.Marshal(before)
	s.audit.Log(audit.Entry{
		ActorID:        actorID(c),
		ActorIP:        c.IP(),
		ActorUserAgent: c.Get("User-Agent"),
		Action:         "user.delete",
		TargetResource: id,
		BeforeState:    beforeJSON,
		Status:         "success",
	})

	if err := s.rbac.Invalidate(c.Context(), id); err != nil {
		s.log.Warn("rbac cache invalidate failed", "user_id", id, "error", err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// handleOffboardUser revokes access in one call: disables the account and
// invalidates its RBAC cache and any live session, with an audit entry
// covering the whole action per the single-command offboarding requirement.
// Per-downstream-app revocation (SCIM deprovisioning) happens in
// internal/provisioning once that phase lands.
func (s *Server) handleOffboardUser(c *fiber.Ctx) error {
	id := c.Params("id")
	before, _ := s.users.Get(c.Context(), id)

	disabled := users.StatusDisabled
	user, err := s.users.Update(c.Context(), id, users.UpdateInput{Status: &disabled})
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(user)
	s.audit.Log(audit.Entry{
		ActorID:        actorID(c),
		ActorIP:        c.IP(),
		ActorUserAgent: c.Get("User-Agent"),
		Action:         "user.offboard",
		TargetResource: id,
		BeforeState:    beforeJSON,
		AfterState:     afterJSON,
		Status:         "success",
	})

	if err := s.rbac.Invalidate(c.Context(), id); err != nil {
		s.log.Warn("rbac cache invalidate failed", "user_id", id, "error", err)
	}

	return c.JSON(user)
}
