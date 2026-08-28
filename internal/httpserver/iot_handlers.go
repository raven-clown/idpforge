package httpserver

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/audit"
	"github.com/raven-clown/idpforge/internal/iot"
)

type createDeviceRequest struct {
	Name       string   `json:"name"`
	DeviceType string   `json:"device_type"`
	Location   string   `json:"location"`
	AllowedIPs []string `json:"allowed_ips"`
}

func (s *Server) handleCreateDevice(c *fiber.Ctx) error {
	var req createDeviceRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" || req.DeviceType == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name and device_type are required")
	}

	device, apiKey, err := s.iot.CreateDevice(c.Context(), req.Name, req.DeviceType, req.Location, req.AllowedIPs)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "could not create device")
	}

	s.audit.Log(audit.Entry{
		ActorID:        actorID(c),
		ActorIP:        c.IP(),
		ActorUserAgent: c.Get("User-Agent"),
		Action:         "iot_device.create",
		TargetResource: device.ID,
		Status:         "success",
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"device":  device,
		"api_key": apiKey, // shown once; store it now, it cannot be retrieved again
	})
}

func (s *Server) handleListDevices(c *fiber.Ctx) error {
	devices, err := s.iot.ListDevices(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not list devices")
	}
	return c.JSON(fiber.Map{"devices": devices})
}

type addCredentialRequest struct {
	CredentialType string `json:"credential_type"`
	CredentialRef  string `json:"credential_ref"`
	Label          string `json:"label"`
}

func (s *Server) handleAddDeviceCredential(c *fiber.Ctx) error {
	userID := c.Params("id")
	var req addCredentialRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.CredentialType == "" || req.CredentialRef == "" {
		return fiber.NewError(fiber.StatusBadRequest, "credential_type and credential_ref are required")
	}

	cred, err := s.iot.AddCredential(c.Context(), userID, req.CredentialType, req.CredentialRef, req.Label)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "could not add credential (already enrolled to someone?)")
	}

	s.audit.Log(audit.Entry{
		ActorID:        actorID(c),
		ActorIP:        c.IP(),
		ActorUserAgent: c.Get("User-Agent"),
		Action:         "device_credential.create",
		TargetResource: userID,
		Status:         "success",
	})

	return c.Status(fiber.StatusCreated).JSON(cred)
}

func (s *Server) handleListDeviceCredentials(c *fiber.Ctx) error {
	creds, err := s.iot.ListCredentials(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not list credentials")
	}
	return c.JSON(fiber.Map{"credentials": creds})
}

func (s *Server) handleDeleteDeviceCredential(c *fiber.Ctx) error {
	if err := s.iot.DeleteCredential(c.Context(), c.Params("cred_id")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not delete credential")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// handleDeviceCheckin is called by reader hardware, authenticated via
// X-Device-Key (requireDeviceKey), not a user session. It resolves the
// scanned credential(s) to a user, records the event, and returns identity
// plus two decision primitives: allowed (an RBAC check against
// "<resource>:access") and already_used_today, for the calling system
// (door controller, canteen POS, attendance kiosk) to apply its own policy.
func (s *Server) handleDeviceCheckin(c *fiber.Ctx) error {
	deviceID := c.Locals("device_id").(string)

	var req iot.CheckinRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if len(req.Credentials) == 0 || req.EventType == "" {
		return fiber.NewError(fiber.StatusBadRequest, "credentials and event_type are required")
	}

	userID, err := s.iot.ResolveUser(c.Context(), req.Credentials)
	status := "matched"
	if err != nil {
		status = "unmatched"
		if err == iot.ErrCredentialMismatch {
			status = "mismatch"
		}
	}

	metadataJSON, _ := json.Marshal(req.Metadata)
	primary := req.Credentials[0]
	eventID, insertErr := s.iot.RecordEvent(c.Context(), iot.Event{
		DeviceID:       deviceID,
		UserID:         userID,
		CredentialType: primary.CredentialType,
		CredentialRef:  primary.CredentialRef,
		EventType:      req.EventType,
		Resource:       req.Resource,
		Metadata:       metadataJSON,
		Status:         status,
		Timestamp:      time.Now().UTC(),
	})
	if insertErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not record event")
	}

	if err != nil {
		return c.JSON(iot.CheckinResponse{
			Allowed: false,
			EventID: eventID,
			Reason:  err.Error(),
		})
	}

	allowed := true
	if req.Resource != "" {
		allowed, _ = s.rbac.HasPermission(c.Context(), userID, req.Resource, "access")
	}
	alreadyToday, _ := s.iot.HasEventToday(c.Context(), userID, req.EventType, req.Resource)

	return c.JSON(iot.CheckinResponse{
		Allowed:      allowed,
		AlreadyToday: alreadyToday,
		UserID:       userID,
		EventID:      eventID,
	})
}

func (s *Server) handleQueryEvents(c *fiber.Ctx) error {
	filter := iot.EventFilter{
		UserID:    c.Query("user_id"),
		DeviceID:  c.Query("device_id"),
		EventType: c.Query("event_type"),
		Resource:  c.Query("resource"),
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

	events, err := s.iot.QueryEvents(c.Context(), filter)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not query events")
	}
	return c.JSON(fiber.Map{"events": events})
}
