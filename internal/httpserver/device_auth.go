package httpserver

import (
	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/netutil"
)

// requireDeviceKey authenticates IoT hardware (readers, kiosks, door
// controllers) via the X-Device-Key header, separate from the user session
// cookie used everywhere else, and enforces that device's own IP/host
// allowlist when one is configured.
func (s *Server) requireDeviceKey(c *fiber.Ctx) error {
	key := c.Get("X-Device-Key")
	if key == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing X-Device-Key header")
	}
	device, err := s.iot.AuthenticateDevice(c.Context(), key)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid device key")
	}
	if !netutil.IPAllowed(c.IP(), device.AllowedIPs) {
		return fiber.NewError(fiber.StatusForbidden, "source IP not allowed for this device")
	}
	c.Locals("device_id", device.ID)
	return c.Next()
}
