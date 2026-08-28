package httpserver

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/audit"
)

func (s *Server) handleQueryAuditLogs(c *fiber.Ctx) error {
	filter := audit.Filter{
		ActorID:   c.Query("actor_id"),
		Action:    c.Query("action"),
		TargetApp: c.Query("target_app"),
		Status:    c.Query("status"),
	}
	if since := c.Query("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			filter.Since = &t
		}
	}
	if until := c.Query("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			filter.Until = &t
		}
	}
	filter.Limit, _ = strconv.Atoi(c.Query("limit", "100"))
	filter.Offset, _ = strconv.Atoi(c.Query("offset", "0"))

	records, err := s.auditReader.Query(c.Context(), filter)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not query audit logs")
	}
	return c.JSON(fiber.Map{"entries": records})
}
