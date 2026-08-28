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
