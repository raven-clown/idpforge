package httpserver

import (
	"bytes"

	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/audit"
	"github.com/raven-clown/idpforge/internal/imageproc"
)

const maxAvatarUploadBytes = 20 << 20 // 20MB raw upload cap, pre-resize

// handleUploadAvatar accepts a raw image body (any common format, up to
// 4K per side), resizes it down to a small sharp thumbnail, and stores it
// via the configured backend (local disk or S3/MinIO).
func (s *Server) handleUploadAvatar(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	body := c.Body()
	if len(body) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "empty upload")
	}
	if len(body) > maxAvatarUploadBytes {
		return fiber.NewError(fiber.StatusRequestEntityTooLarge, "image too large")
	}

	processed, contentType, err := imageproc.ProcessAvatar(bytes.NewReader(body))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not process image: "+err.Error())
	}

	ext := ".jpg"
	if contentType == "image/png" {
		ext = ".png"
	}
	key := userID + ext

	url, err := s.storage.Put(c.Context(), key, bytes.NewReader(processed), int64(len(processed)), contentType)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not store image")
	}

	if err := s.users.SetAvatar(c.Context(), userID, url); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not save avatar reference")
	}

	s.audit.Log(audit.Entry{
		ActorID:        userID,
		ActorIP:        c.IP(),
		ActorUserAgent: c.Get("User-Agent"),
		Action:         "user.avatar_upload",
		TargetResource: userID,
		Status:         "success",
	})

	return c.JSON(fiber.Map{"avatar_url": url})
}
