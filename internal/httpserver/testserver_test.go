package httpserver

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/raven-clown/idpforge/internal/announcements"
	"github.com/raven-clown/idpforge/internal/apiclient"
	"github.com/raven-clown/idpforge/internal/audit"
	"github.com/raven-clown/idpforge/internal/auth/oidc"
	"github.com/raven-clown/idpforge/internal/bootstrap"
	"github.com/raven-clown/idpforge/internal/cache"
	"github.com/raven-clown/idpforge/internal/captcha"
	"github.com/raven-clown/idpforge/internal/config"
	"github.com/raven-clown/idpforge/internal/health"
	"github.com/raven-clown/idpforge/internal/iot"
	"github.com/raven-clown/idpforge/internal/mfa"
	"github.com/raven-clown/idpforge/internal/rbac"
	"github.com/raven-clown/idpforge/internal/storage"
	"github.com/raven-clown/idpforge/internal/testutil"
	"github.com/raven-clown/idpforge/internal/users"
	"github.com/raven-clown/idpforge/internal/webauthn"
)

// testHarness bundles a real *Server (wired the same way cmd/server/main.go
// wires one, minus the background jobs) with the repositories tests need
// direct access to -- e.g. to seed data or read back state the API
// wouldn't otherwise expose.
type testHarness struct {
	srv     *Server
	cfg     *config.Config
	users   *users.Repository
	rbacAdm *rbac.Admin
}

// newTestServer builds a fully wired Server against a fresh in-memory
// SQLite DB, with a bootstrap admin account (username "admin", password
// "AdminBoot123!"). cfgOverride, if non-nil, is applied to the default
// test config before the server is built.
func newTestServer(t *testing.T, cfgOverride func(*config.Config)) *testHarness {
	t.Helper()

	database := testutil.OpenTestDB(t)
	ctx := context.Background()

	cfg := &config.Config{
		Env: "development",
		HTTP: config.HTTPConfig{
			ListenAddr: ":0",
			BaseURL:    "http://localhost:8080",
		},
		Captcha: config.CaptchaConfig{Provider: "none"},
		OIDC: config.OIDCConfig{
			Issuer:          "http://localhost:8080",
			SigningKeyPath:  filepath.Join(t.TempDir(), "signing-key.pem"),
			AccessTokenTTL:  15 * time.Minute,
			IDTokenTTL:      15 * time.Minute,
			RefreshTokenTTL: 30 * 24 * time.Hour,
		},
		Storage: config.StorageConfig{Backend: "local", LocalDir: t.TempDir()},
		Bootstrap: config.BootstrapConfig{
			AdminUsername: "admin",
			AdminEmail:    "admin@localhost",
			AdminPassword: "AdminBoot123!",
		},
		PasswordPolicy: config.PasswordPolicyConfig{
			MinLength: 8, RequireUppercase: true, RequireLowercase: true, RequireNumber: true, RequireSpecial: true,
		},
		DefaultPassword: "Welcome123!",
	}
	if cfgOverride != nil {
		cfgOverride(cfg)
	}

	c := cache.NewMemory()

	userRepo := users.NewRepository(database)
	resolver := rbac.NewResolver(database, c)
	rbacAdmin := rbac.NewAdmin(database, resolver)
	iotRepo := iot.NewRepository(database)
	apiClientRepo := apiclient.NewRepository(database)
	announceRepo := announcements.NewRepository(database)

	if err := bootstrap.Run(ctx, database, userRepo, rbacAdmin, cfg.Bootstrap, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	store, err := storage.New(cfg.Storage)
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}

	healthChecker := health.NewChecker()

	auditWriter := audit.NewWriter(database, 100, 10, time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	auditWriter.Run(ctx)
	t.Cleanup(auditWriter.Stop)
	auditReader := audit.NewReader(database)

	keys, err := oidc.LoadOrGenerateKey(cfg.OIDC.SigningKeyPath)
	if err != nil {
		t.Fatalf("LoadOrGenerateKey: %v", err)
	}
	clientStore := oidc.NewClientStore(database)
	provider := oidc.NewProvider(cfg.OIDC, keys, clientStore, c, resolver)

	waStore := webauthn.NewCredentialStore(database)
	waService, err := webauthn.NewService("localhost", "IdpForge Test", cfg.HTTP.BaseURL, waStore, c)
	if err != nil {
		t.Fatalf("webauthn.NewService: %v", err)
	}

	mfaService := mfa.NewService(database, "IdpForge Test")
	captchaVerifier := captcha.New(cfg.Captcha.Provider, cfg.Captcha.SecretKey)

	srv := New(Deps{
		Config:      cfg,
		Users:       userRepo,
		RBAC:        resolver,
		RBACAdm:     rbacAdmin,
		Audit:       auditWriter,
		AuditReader: auditReader,
		OIDC:        provider,
		WebAuthn:    waService,
		MFA:         mfaService,
		Captcha:     captchaVerifier,
		Storage:     store,
		Health:      healthChecker,
		IoT:         iotRepo,
		APIClients:  apiClientRepo,
		Announce:    announceRepo,
		Cache:       c,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		Version:     "test",
	})

	return &testHarness{srv: srv, cfg: cfg, users: userRepo, rbacAdm: rbacAdmin}
}
