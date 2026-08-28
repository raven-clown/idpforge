package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/raven-clown/idpforge/internal/audit"
	"github.com/raven-clown/idpforge/internal/auth/oidc"
	"github.com/raven-clown/idpforge/internal/cache"
	"github.com/raven-clown/idpforge/internal/captcha"
	"github.com/raven-clown/idpforge/internal/config"
	"github.com/raven-clown/idpforge/internal/db"
	"github.com/raven-clown/idpforge/internal/httpserver"
	"github.com/raven-clown/idpforge/internal/mfa"
	"github.com/raven-clown/idpforge/internal/rbac"
	"github.com/raven-clown/idpforge/internal/service"
	"github.com/raven-clown/idpforge/internal/users"
	"github.com/raven-clown/idpforge/internal/webauthn"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load failed", "error", err)
		os.Exit(1)
	}

	if err := run(cfg, logger); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, logger *slog.Logger) error {
	ctx := context.Background()

	database, err := db.Open(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer database.Close()
	logger.Info("connected to database", "driver", cfg.DB.Driver, "dsn", cfg.DB.MaskedDSN())

	if err := database.Migrate(ctx); err != nil {
		return err
	}
	logger.Info("migrations applied")

	var c cache.Cache
	if cfg.Redis.Enabled {
		redisCache := cache.NewRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
		if err := redisCache.Ping(ctx); err != nil {
			logger.Warn("redis unreachable, falling back to in-memory cache", "error", err)
			c = cache.NewMemory()
		} else {
			c = redisCache
			logger.Info("connected to redis", "addr", cfg.Redis.Addr)
		}
	} else {
		c = cache.NewMemory()
		logger.Info("redis disabled, using in-memory cache (single-node only)")
	}

	userRepo := users.NewRepository(database)
	resolver := rbac.NewResolver(database, c)

	auditWriter := audit.NewWriter(database, cfg.Audit.QueueSize, cfg.Audit.BatchSize, cfg.Audit.FlushInterval, logger)
	auditCtx, auditCancel := context.WithCancel(ctx)
	go auditWriter.Run(auditCtx)

	keys, err := oidc.LoadOrGenerateKey(cfg.OIDC.SigningKeyPath)
	if err != nil {
		auditCancel()
		return err
	}
	clientStore := oidc.NewClientStore(database)
	provider := oidc.NewProvider(cfg.OIDC, keys, clientStore, c, resolver)

	waStore := webauthn.NewCredentialStore(database)
	waService, err := webauthn.NewService(rpIDFromBaseURL(cfg.HTTP.BaseURL), "IdpForge", cfg.HTTP.BaseURL, waStore, c)
	if err != nil {
		auditCancel()
		return err
	}

	mfaService := mfa.NewService(database, "IdpForge")
	captchaVerifier := captcha.New(cfg.Captcha.Provider, cfg.Captcha.SecretKey)

	srv := httpserver.New(httpserver.Deps{
		Config:   cfg,
		Users:    userRepo,
		RBAC:     resolver,
		Audit:    auditWriter,
		OIDC:     provider,
		WebAuthn: waService,
		MFA:      mfaService,
		Captcha:  captchaVerifier,
		Cache:    c,
		Logger:   logger,
	})

	return service.Run(
		func(ctx context.Context) error {
			logger.Info("listening", "addr", cfg.HTTP.ListenAddr)
			return srv.Listen(cfg.HTTP.ListenAddr)
		},
		func() {
			logger.Info("shutting down")
			_ = srv.ShutdownWithTimeout(10 * time.Second)
			auditWriter.Stop()
			auditCancel()
			_ = c.Close()
		},
	)
}

// rpIDFromBaseURL strips scheme/port from the configured base URL to get a
// WebAuthn Relying Party ID (must be a bare hostname).
func rpIDFromBaseURL(base string) string {
	host := base
	for _, prefix := range []string{"https://", "http://"} {
		if len(host) > len(prefix) && host[:len(prefix)] == prefix {
			host = host[len(prefix):]
			break
		}
	}
	for i, ch := range host {
		if ch == ':' || ch == '/' {
			return host[:i]
		}
	}
	return host
}
