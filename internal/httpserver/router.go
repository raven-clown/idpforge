package httpserver

func (s *Server) registerRoutes() {
	s.app.Get("/healthz", s.handleHealth)
	s.app.Get("/metrics", s.handleMetrics)

	s.app.Get("/.well-known/openid-configuration", s.handleDiscovery)
	s.app.Get("/oauth2/jwks", s.handleJWKS)
	s.app.Get("/oauth2/authorize", s.handleAuthorize)
	s.app.Post("/oauth2/token", s.handleToken)
	s.app.Get("/oauth2/userinfo", s.handleUserinfo)

	s.app.Post("/api/v1/login", s.handleLogin)
	s.app.Post("/api/v1/logout", s.handleLogout)

	api := s.app.Group("/api/v1", requireSession(s.sessions))

	u := api.Group("/users")
	u.Post("/", s.handleCreateUser)
	u.Get("/", s.handleListUsers)
	u.Get("/:id", s.handleGetUser)
	u.Patch("/:id", s.handleUpdateUser)
	u.Delete("/:id", s.handleDeleteUser)
	u.Post("/:id/offboard", s.handleOffboardUser)

	wa := api.Group("/webauthn")
	wa.Post("/register/begin", s.handleWebAuthnRegisterBegin)
	wa.Post("/register/finish", s.handleWebAuthnRegisterFinish)
	wa.Post("/login/begin", s.handleWebAuthnLoginBegin)
	wa.Post("/login/finish", s.handleWebAuthnLoginFinish)

	m := api.Group("/mfa")
	m.Post("/enroll", s.handleMFAEnroll)
	m.Post("/confirm", s.handleMFAConfirm)
	m.Post("/verify", s.handleMFAVerify)

	fa := s.app.Group("/forwardauth")
	fa.Get("/", s.handleForwardAuth)
}
