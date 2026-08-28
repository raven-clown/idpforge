package httpserver

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// handleMetricsHistory backs the admin UI's usage graphs: cumulative
// snapshots since N days ago, oldest first, for the client to diff into
// per-interval deltas.
func (s *Server) handleMetricsHistory(c *fiber.Ctx) error {
	days, _ := strconv.Atoi(c.Query("days", "30"))
	if days <= 0 || days > 365 {
		days = 30
	}
	since := time.Now().UTC().AddDate(0, 0, -days)

	snapshots, err := s.metricsHist.Since(c.Context(), since)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not load metrics history")
	}
	return c.JSON(fiber.Map{"snapshots": snapshots})
}
