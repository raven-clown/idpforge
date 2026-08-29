package httpserver

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/raven-clown/idpforge/internal/announcements"
	"github.com/raven-clown/idpforge/internal/audit"
	"github.com/raven-clown/idpforge/internal/cache"
	"github.com/raven-clown/idpforge/internal/metrics"
)

// wsClusterChannel is the Redis pub/sub channel realtimeHub uses to fan
// events out across every instance in a multi-instance deployment. Only
// active when Redis is enabled -- without it, a broadcast only ever
// reaches browsers connected to the same instance that handled the
// request that triggered it.
const wsClusterChannel = "idpforge:ws:broadcast"

// wsEnvelope wraps a broadcast payload for the cluster channel. originID
// lets every instance (including the publisher) ignore its own message
// when it arrives back via its own subscription -- the publisher already
// delivered it to its local connections directly, at publish time.
type wsEnvelope struct {
	OriginID  string `json:"origin_id"`
	AuditOnly bool   `json:"audit_only"`
	Payload   []byte `json:"payload"`
}

// realtimeHub fans live audit events and announcements out to connected
// sessions over WebSocket, so the notification bell and audit log page
// update without polling. It holds no state beyond the current connection
// set: audit.Writer's batched DB insert and the announcements table remain
// the durable sources of truth, this is a best-effort live feed on top of
// them.
//
// Audit events carry sensitive detail (actor IPs, target resources), so
// they're only ever sent to a connection whose session held audit:read at
// connect time. Announcements go to every connection: they're meant to be
// seen by everyone signed in.
type realtimeHub struct {
	mu         sync.RWMutex
	clients    map[*websocket.Conn]bool // value: canSeeAudit
	instanceID string
	redis      *cache.RedisCache // nil unless Redis is enabled
}

// newRealtimeHub starts the hub. Pass a non-nil redisCache to fan
// broadcasts out across every instance sharing that Redis; pass nil for a
// single-instance deployment, where broadcasts only ever reach browsers
// connected to this instance.
func newRealtimeHub(redisCache *cache.RedisCache) *realtimeHub {
	h := &realtimeHub{
		clients:    make(map[*websocket.Conn]bool),
		instanceID: uuid.NewString(),
		redis:      redisCache,
	}
	if redisCache != nil {
		go h.subscribeCluster()
	}
	return h
}

func (h *realtimeHub) subscribeCluster() {
	msgs, cancel := h.redis.Subscribe(context.Background(), wsClusterChannel)
	defer cancel()
	for raw := range msgs {
		var env wsEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			continue
		}
		if env.OriginID == h.instanceID {
			continue // this instance already delivered it locally at publish time
		}
		h.deliverLocal(env.Payload, env.AuditOnly)
	}
}

func (h *realtimeHub) add(c *websocket.Conn, canSeeAudit bool) {
	h.mu.Lock()
	h.clients[c] = canSeeAudit
	h.mu.Unlock()
	metrics.WebSocketConnections.Inc()
}

func (h *realtimeHub) remove(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	metrics.WebSocketConnections.Dec()
}

// broadcast delivers msg to this instance's own connections immediately,
// then (when clustered) publishes it for every other instance to relay to
// theirs.
func (h *realtimeHub) broadcast(msg []byte, auditOnly bool) {
	h.deliverLocal(msg, auditOnly)
	if h.redis == nil {
		return
	}
	payload, err := json.Marshal(wsEnvelope{OriginID: h.instanceID, AuditOnly: auditOnly, Payload: msg})
	if err != nil {
		return
	}
	_ = h.redis.Publish(context.Background(), wsClusterChannel, payload)
}

func (h *realtimeHub) deliverLocal(msg []byte, auditOnly bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c, canSeeAudit := range h.clients {
		if auditOnly && !canSeeAudit {
			continue
		}
		_ = c.WriteMessage(websocket.TextMessage, msg)
	}
}

type wsAuditEvent struct {
	Type      string `json:"type"`
	ActorID   string `json:"actor_id,omitempty"`
	Action    string `json:"action"`
	Target    string `json:"target_resource,omitempty"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// onAuditLog is wired as audit.Writer's OnLog hook: every entry queued for
// the DB is also pushed live to connected clients immediately, ahead of
// the writer's own flush interval.
func (h *realtimeHub) onAuditLog(e audit.Entry) {
	payload, err := json.Marshal(wsAuditEvent{
		Type:      "audit_log",
		ActorID:   e.ActorID,
		Action:    e.Action,
		Target:    e.TargetResource,
		Status:    e.Status,
		Timestamp: e.Timestamp.Format(time.RFC3339),
	})
	if err != nil {
		return
	}
	h.broadcast(payload, true)
}

type wsAnnouncementEvent struct {
	Type         string                     `json:"type"`
	Announcement announcements.Announcement `json:"announcement"`
}

func (h *realtimeHub) onAnnouncement(a announcements.Announcement) {
	payload, err := json.Marshal(wsAnnouncementEvent{Type: "announcement", Announcement: a})
	if err != nil {
		return
	}
	h.broadcast(payload, false)
}

// NotifySystem persists and broadcasts a system-generated announcement --
// e.g. the update-checker's "a new version is available" notice -- through
// the same channel an admin-authored one uses.
func (s *Server) NotifySystem(ctx context.Context, message string, level announcements.Level) error {
	a, err := s.announce.Create(ctx, message, level, "")
	if err != nil {
		return err
	}
	s.hub.onAnnouncement(*a)
	return nil
}

// wsAuthContext resolves whether the connecting session can see audit
// events, once, before the upgrade -- there's no per-message auth on a
// WebSocket connection, so this is decided for the lifetime of the
// connection. Must run after authenticateAny.
func (s *Server) wsAuthContext(c *fiber.Ctx) error {
	canSeeAudit := false
	if userID, ok := c.Locals("user_id").(string); ok && userID != "" {
		if allowed, err := s.rbac.HasPermission(c.Context(), userID, "audit", "read"); err == nil {
			canSeeAudit = allowed
		}
	}
	c.Locals("can_see_audit", canSeeAudit)
	return c.Next()
}

// wsUpgradeGuard rejects a plain HTTP GET on the WebSocket route with a
// normal error response instead of hanging; the actual upgrade happens in
// the websocket.New handler registered after this one.
func wsUpgradeGuard(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// handleWS holds the connection open and fans out realtimeHub broadcasts.
// The client is never expected to send anything; the read loop exists only
// to notice a close/error so the connection gets cleaned up promptly.
func (s *Server) handleWS(c *websocket.Conn) {
	canSeeAudit, _ := c.Locals("can_see_audit").(bool)
	s.hub.add(c, canSeeAudit)
	defer s.hub.remove(c)

	for {
		if _, _, err := c.ReadMessage(); err != nil {
			return
		}
	}
}
