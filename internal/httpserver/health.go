package httpserver

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Server) handleHealth(c *fiber.Ctx) error {
	results, ok := s.health.Run(c.Context())
	status := fiber.StatusOK
	if !ok {
		status = fiber.StatusServiceUnavailable
	}
	return c.Status(status).JSON(fiber.Map{"status": statusLabel(ok), "checks": results})
}

func statusLabel(ok bool) string {
	if ok {
		return "ok"
	}
	return "degraded"
}

func (s *Server) handleMetrics(c *fiber.Ctx) error {
	return adaptor.HTTPHandler(promhttp.Handler())(c)
}
