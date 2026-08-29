package httpserver

import "github.com/gofiber/contrib/websocket"

func (s *Server) registerRoutes() {
	s.app.Get("/healthz", s.handleHealth)
	s.app.Get("/metrics", s.handleMetrics)

	discovery := s.app.Group("/.well-known", s.rateLimitGlobal)
	discovery.Get("/openid-configuration", s.handleDiscovery)

	oauth := s.app.Group("/oauth2", s.rateLimitGlobal)
	oauth.Get("/jwks", s.handleJWKS)
	oauth.Get("/authorize", s.handleAuthorize)
	oauth.Post("/token", s.handleToken)
	oauth.Get("/userinfo", s.handleUserinfo)

	s.app.Post("/api/v1/login", s.rateLimitLogin, s.handleLogin)
	s.app.Post("/api/v1/logout", s.handleLogout)

	// api accepts a user session OR a scoped API client (X-API-Key), so a
	// granted token can call the same admin API a logged-in admin would,
	// restricted to whatever scopes it was given.
	api := s.app.Group("/api/v1", s.authenticateAny, s.rateLimitGlobal)

	u := api.Group("/users")
	u.Post("/", s.requirePermission("users", "manage"), s.handleCreateUser)
	u.Get("/", s.requirePermission("users", "read"), s.handleListUsers)
	u.Get("/:id", s.requirePermission("users", "read"), s.handleGetUser)
	u.Patch("/:id", s.requirePermission("users", "manage"), s.handleUpdateUser)
	u.Delete("/:id", s.requirePermission("users", "manage"), s.handleDeleteUser)
	u.Post("/:id/offboard", s.requirePermission("users", "manage"), s.handleOffboardUser)
	u.Post("/:id/reset-password", s.requirePermission("users", "manage"), s.handleResetUserPassword)
	u.Post("/:id/device-credentials", s.requirePermission("users", "manage"), s.handleAddDeviceCredential)
	u.Get("/:id/device-credentials", s.requirePermission("users", "read"), s.handleListDeviceCredentials)
	u.Delete("/:id/device-credentials/:cred_id", s.requirePermission("users", "manage"), s.handleDeleteDeviceCredential)

	// Self-service only: a real browser/human ceremony (WebAuthn), the
	// caller's own MFA secret, or the caller's own profile picture. Not
	// something a scoped API client token acts on for someone else.
	self := api.Group("/", requireUserActor)
	self.Post("/users/:id/avatar", s.handleUploadAvatar)
	self.Get("/me", s.handleMe)
	self.Post("/change-password", s.handleChangePassword)

	wa := self.Group("/webauthn")
	wa.Post("/register/begin", s.handleWebAuthnRegisterBegin)
	wa.Post("/register/finish", s.handleWebAuthnRegisterFinish)
	wa.Post("/login/begin", s.handleWebAuthnLoginBegin)
	wa.Post("/login/finish", s.handleWebAuthnLoginFinish)

	m := self.Group("/mfa")
	m.Post("/enroll", s.handleMFAEnroll)
	m.Post("/confirm", s.handleMFAConfirm)
	m.Post("/verify", s.handleMFAVerify)
	m.Post("/disable", s.handleMFADisable)

	rbacGroup := api.Group("/rbac", s.requirePermission("rbac", "manage"))
	rbacGroup.Post("/roles", s.handleCreateRole)
	rbacGroup.Get("/roles", s.handleListRoles)
	rbacGroup.Get("/roles/:id", s.handleGetRole)
	rbacGroup.Get("/roles/:id/permissions", s.handleListRolePermissions)
	rbacGroup.Delete("/roles/:id", s.handleDeleteRole)
	rbacGroup.Post("/roles/:id/permissions", s.handleGrantPermissionToRole)
	rbacGroup.Delete("/roles/:id/permissions/:permission_id", s.handleRevokePermissionFromRole)
	rbacGroup.Post("/permissions", s.handleCreatePermission)
	rbacGroup.Get("/permissions", s.handleListPermissions)
	rbacGroup.Post("/groups", s.handleCreateGroup)
	rbacGroup.Get("/groups", s.handleListGroups)
	rbacGroup.Post("/groups/:id/roles", s.handleAssignRoleToGroup)
	rbacGroup.Delete("/groups/:id/roles/:role_id", s.handleRemoveRoleFromGroup)
	rbacGroup.Post("/groups/:id/users/:user_id", s.handleAddUserToGroup)
	rbacGroup.Delete("/groups/:id/users/:user_id", s.handleRemoveUserFromGroup)
	rbacGroup.Get("/users/:id/roles", s.handleListUserRoles)
	rbacGroup.Post("/users/:id/roles", s.handleAssignRoleToUser)
	rbacGroup.Delete("/users/:id/roles/:role_id", s.handleRemoveRoleFromUser)

	iotAdmin := api.Group("/iot", s.requirePermission("iot", "manage"))
	iotAdmin.Post("/devices", s.handleCreateDevice)
	iotAdmin.Get("/devices", s.handleListDevices)
	iotEvents := api.Group("/iot", s.requirePermission("iot", "read"))
	iotEvents.Get("/events", s.handleQueryEvents)

	clientsAdmin := api.Group("/api-clients", s.requirePermission("api_clients", "manage"))
	clientsAdmin.Post("/", s.handleCreateAPIClient)
	clientsAdmin.Get("/", s.handleListAPIClients)
	clientsAdmin.Delete("/:id", s.handleDeleteAPIClient)

	api.Get("/audit-logs", s.requirePermission("audit", "read"), s.handleQueryAuditLogs)
	api.Get("/metrics/history", s.requirePermission("metrics", "read"), s.handleMetricsHistory)
	api.Get("/settings", s.requirePermission("settings", "read"), s.handleGetSettings)

	// Announcements: any signed-in user can read them, only an admin
	// (or the update-checker, server-side) can post one.
	api.Get("/announcements", s.handleListAnnouncements)
	api.Post("/announcements", s.requirePermission("announcements", "manage"), s.handleCreateAnnouncement)

	// Realtime feed (audit events + announcements) over WebSocket, open to
	// any signed-in user -- announcements are for everyone, audit events
	// are filtered per-connection inside the hub based on audit:read.
	// Browser WebSocket clients can't set custom headers, so this only
	// supports session-cookie auth, not X-API-Key.
	api.Get("/ws", s.wsAuthContext, wsUpgradeGuard, websocket.New(s.handleWS))

	// Device-authenticated (X-Device-Key), not a user session: hardware
	// check-in endpoint for badge/face/fingerprint readers, door
	// controllers, kiosks. Deliberately NOT under /iot -- that path is also
	// the SPA's IoT admin page, and Fiber's prefix-matched Use middleware
	// (which is what Group(prefix, handlers...) registers) would otherwise
	// intercept every browser request to the admin page and demand an
	// X-Device-Key before the page ever got a chance to render.
	iotDevice := s.app.Group("/device/v1", s.requireDeviceKey)
	iotDevice.Post("/checkin", s.handleDeviceCheckin)

	// API-key authenticated (X-API-Key), not a user session: the simple,
	// field-filtered, per-client-rate-limited path for apps or AI services
	// that just need "verify a login" or "look up a user", with no RBAC
	// scopes required. For full read/write access to the real admin API,
	// grant scopes and use /api/v1 instead. Creating a user is the one
	// write action available here (for a provisioning/HR integration),
	// and it does require the users:manage scope, checked inside the
	// handler -- everything else on this path stays scope-free.
	external := s.app.Group("/external/v1", s.requireAPIClient)
	external.Post("/login", s.handleExternalLogin)
	external.Get("/users/:id", s.handleExternalGetUser)
	external.Post("/users", s.handleExternalCreateUser)

	fa := s.app.Group("/forwardauth")
	fa.Get("/", s.handleForwardAuth)

	if err := s.registerSPA(); err != nil {
		panic("failed to mount admin SPA: " + err.Error())
	}
}
