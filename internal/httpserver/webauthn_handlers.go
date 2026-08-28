package httpserver

import "github.com/gofiber/fiber/v2"

func (s *Server) handleWebAuthnRegisterBegin(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	options, err := s.webauthn.BeginRegistration(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(options)
}

func (s *Server) handleWebAuthnRegisterFinish(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	if err := s.webauthn.FinishRegistration(c.Context(), userID, c.Body()); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"status": "registered"})
}

func (s *Server) handleWebAuthnLoginBegin(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	options, err := s.webauthn.BeginLogin(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(options)
}

func (s *Server) handleWebAuthnLoginFinish(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	if err := s.webauthn.FinishLogin(c.Context(), userID, c.Body()); err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}
	return c.JSON(fiber.Map{"status": "verified"})
}
