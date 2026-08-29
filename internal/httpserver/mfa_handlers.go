package httpserver

import "github.com/gofiber/fiber/v2"

func (s *Server) handleMFAEnroll(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	user, err := s.users.Get(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	secret, url, err := s.mfa.Enroll(c.Context(), userID, user.Username)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"secret": secret, "otpauth_url": url})
}

type mfaCodeRequest struct {
	Code string `json:"code"`
}

func (s *Server) handleMFAConfirm(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var req mfaCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := s.mfa.Confirm(c.Context(), userID, req.Code); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid code")
	}
	return c.JSON(fiber.Map{"status": "enabled"})
}

// handleMFADisable requires the caller's current password, the same
// safeguard the self-service change-password endpoint uses, since turning
// off MFA lowers the account's own security bar.
func (s *Server) handleMFADisable(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var req struct {
		CurrentPassword string `json:"current_password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	user, err := s.users.Get(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}
	hash, err := s.users.PasswordHash(c.Context(), user.Username)
	if err != nil || hash == "" || !s.users.VerifyPassword(hash, req.CurrentPassword) {
		return fiber.NewError(fiber.StatusUnauthorized, "current password is incorrect")
	}

	if err := s.mfa.Disable(c.Context(), userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not disable MFA")
	}
	return c.JSON(fiber.Map{"status": "disabled"})
}

func (s *Server) handleMFAVerify(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var req mfaCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	ok, err := s.mfa.Verify(c.Context(), userID, req.Code)
	if err != nil || !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid code")
	}
	return c.JSON(fiber.Map{"status": "verified"})
}
