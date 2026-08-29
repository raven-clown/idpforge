package httpserver

import (
	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/announcements"
)

type createAnnouncementRequest struct {
	Message string `json:"message"`
	Level   string `json:"level"`
}

// handleCreateAnnouncement lets an admin broadcast a message to everyone
// signed in right now (over WebSocket) and persists it so anyone who
// connects later still sees it. The same path the update-checker uses for
// system-generated "new version available" notices.
func (s *Server) handleCreateAnnouncement(c *fiber.Ctx) error {
	var req createAnnouncementRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Message == "" {
		return fiber.NewError(fiber.StatusBadRequest, "message is required")
	}
	level := announcements.Level(req.Level)
	switch level {
	case announcements.LevelInfo, announcements.LevelWarning, announcements.LevelCritical:
	case "":
		level = announcements.LevelInfo
	default:
		return fiber.NewError(fiber.StatusBadRequest, "level must be info, warning, or critical")
	}

	a, err := s.announce.Create(c.Context(), req.Message, level, actorID(c))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not create announcement")
	}
	s.hub.onAnnouncement(*a)

	return c.Status(fiber.StatusCreated).JSON(a)
}

// handleListAnnouncements is available to any authenticated user (not
// gated behind an admin permission): announcements are meant to be seen by
// everyone, not managed by everyone.
func (s *Server) handleListAnnouncements(c *fiber.Ctx) error {
	list, err := s.announce.List(c.Context(), 20)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not list announcements")
	}
	return c.JSON(fiber.Map{"announcements": list})
}
