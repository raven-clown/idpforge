package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/raven-clown/idpforge/internal/announcements"
	"github.com/raven-clown/idpforge/internal/apiclient"
	"github.com/raven-clown/idpforge/internal/audit"
	"github.com/raven-clown/idpforge/internal/auth/oidc"
	"github.com/raven-clown/idpforge/internal/bootstrap"
	"github.com/raven-clown/idpforge/internal/cache"
	"github.com/raven-clown/idpforge/internal/captcha"
	"github.com/raven-clown/idpforge/internal/config"
	"github.com/raven-clown/idpforge/internal/db"
	"github.com/raven-clown/idpforge/internal/health"
	"github.com/raven-clown/idpforge/internal/httpserver"
	"github.com/raven-clown/idpforge/internal/iot"
	"github.com/raven-clown/idpforge/internal/leaderlease"
	"github.com/raven-clown/idpforge/internal/metrics"
	"github.com/raven-clown/idpforge/internal/mfa"
	"github.com/raven-clown/idpforge/internal/rbac"
	"github.com/raven-clown/idpforge/internal/service"
	"github.com/raven-clown/idpforge/internal/storage"
	"github.com/raven-clown/idpforge/internal/updatecheck"
	"github.com/raven-clown/idpforge/internal/users"
	"github.com/raven-clown/idpforge/internal/webauthn"
)

// version is set at build time via -ldflags "-X main.version=...", from
// the git tag on a release build (see build.ps1 / Makefile). Left as "dev"
// for a local, non-release build, which the update-checker treats as
// nothing to compare against.
var version = "dev"

const updateCheckRepoOwner = "raven-clown"
const updateCheckRepoName = "idpforge"

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
	rbacAdmin := rbac.NewAdmin(database, resolver)
	iotRepo := iot.NewRepository(database)
	apiClientRepo := apiclient.NewRepository(database)
	announceRepo := announcements.NewRepository(database)

	if err := bootstrap.Run(ctx, database, userRepo, rbacAdmin, cfg.Bootstrap, logger); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}

	store, err := storage.New(cfg.Storage)
	if err != nil {
		return err
	}

	healthChecker := health.NewChecker()
	healthChecker.Register("database", func(ctx context.Context) error { return database.PingContext(ctx) })
	healthChecker.Register("cache", func(ctx context.Context) error {
		if pinger, ok := c.(interface{ Ping(context.Context) error }); ok {
			return pinger.Ping(ctx)
		}
		return nil
	})
	healthChecker.Register("disk_space", health.DiskSpaceCheck(cfg.Paths.DataDir, 100<<20)) // 100MB floor

	auditWriter := audit.NewWriter(database, cfg.Audit.QueueSize, cfg.Audit.BatchSize, cfg.Audit.FlushInterval, logger)
	auditCtx, auditCancel := context.WithCancel(ctx)
	auditWriter.Run(auditCtx)
	auditReader := audit.NewReader(database)

	// lease makes the periodic background jobs below safe to run on every
	// instance in a multi-instance deployment: each tick, only the current
	// leader for that job actually does the work. A single instance always
	// wins its own lease uncontested, so this costs nothing meaningful in
	// the common single-node case.
	lease := leaderlease.New(database)

	metricsHistory := metrics.NewHistory(database)
	metricsSamplerCtx, stopMetricsSampler := context.WithCancel(ctx)
	defer stopMetricsSampler()
	go runMetricsSampler(metricsSamplerCtx, metricsHistory, cfg.Storage, lease, logger)

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
		Config:      cfg,
		Users:       userRepo,
		RBAC:        resolver,
		RBACAdm:     rbacAdmin,
		Audit:       auditWriter,
		AuditReader: auditReader,
		MetricsHist: metricsHistory,
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
		Logger:      logger,
		Version:     version,
	})

	bgCtx, stopBg := context.WithCancel(ctx)
	defer stopBg()
	if cfg.UpdateCheck.Enabled {
		go runUpdateChecker(bgCtx, srv, cfg.UpdateCheck.Interval, version, lease, logger)
	}
	go runHealthAlerts(bgCtx, srv, healthChecker, lease, logger)

	return service.Run(
		func(ctx context.Context) error {
			logger.Info("listening", "addr", cfg.HTTP.ListenAddr, "version", version)
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

// runUpdateChecker polls GitHub Releases for a newer IdpForge version than
// the one currently running, so a self-hosted operator finds out in the
// admin UI instead of having to remember to check. It checks once on
// startup, then on the configured interval; a failed check (offline, rate
// limited) just tries again next interval.
func runUpdateChecker(ctx context.Context, srv *httpserver.Server, interval time.Duration, currentVersion string, lease *leaderlease.Lease, logger *slog.Logger) {
	check := func() {
		if leader, err := lease.TryAcquire(ctx, "update-checker", interval*2); err != nil || !leader {
			return
		}
		rel, err := updatecheck.LatestRelease(ctx, updateCheckRepoOwner, updateCheckRepoName)
		if err != nil {
			logger.Warn("update check failed", "error", err)
			return
		}
		if !updatecheck.IsNewer(currentVersion, rel.TagName) {
			return
		}
		msg := fmt.Sprintf("IdpForge %s is available (you're running %s). %s", rel.TagName, currentVersion, rel.HTMLURL)
		if err := srv.NotifySystem(ctx, msg, announcements.LevelInfo); err != nil {
			logger.Warn("update check: could not post announcement", "error", err)
			return
		}
		logger.Info("update available", "latest", rel.TagName, "current", currentVersion)
	}

	check()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}

// healthAlertInterval controls how often runHealthAlerts re-runs the
// registered health checks. Faster than the update check on purpose: a
// database or cache outage is worth surfacing within a minute, not a day.
const healthAlertInterval = 30 * time.Second

// runHealthAlerts posts a system announcement whenever a registered health
// check (database, cache, disk space) changes state, so any operational
// problem shows up right in the same notification bell as everything
// else, instead of only being visible on /healthz to whoever remembers to
// poll it.
func runHealthAlerts(ctx context.Context, srv *httpserver.Server, checker *health.Checker, lease *leaderlease.Lease, logger *slog.Logger) {
	prevOK := make(map[string]bool)

	// Seed state without alerting: a check that's already failing when the
	// process starts isn't a new transition worth announcing. Every
	// instance seeds its own local prevOK map (harmless even though only
	// the leader will go on to post announcements) so whichever instance
	// wins the lease first already has a correct baseline.
	seed, _ := checker.Run(ctx)
	for _, r := range seed {
		prevOK[r.Name] = r.OK
	}

	ticker := time.NewTicker(healthAlertInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if leader, err := lease.TryAcquire(ctx, "health-alerts", healthAlertInterval*2); err != nil || !leader {
				continue
			}
			results, _ := checker.Run(ctx)
			for _, r := range results {
				was := prevOK[r.Name]
				prevOK[r.Name] = r.OK
				if was == r.OK {
					continue
				}

				level := announcements.LevelInfo
				msg := fmt.Sprintf("Health check %q has recovered.", r.Name)
				if !r.OK {
					level = announcements.LevelCritical
					msg = fmt.Sprintf("Health check %q is failing: %s", r.Name, r.Error)
				}
				if err := srv.NotifySystem(ctx, msg, level); err != nil {
					logger.Warn("health alert: could not post announcement", "error", err)
				}
			}
		}
	}
}

// runMetricsSampler records the current cumulative counters every 10
// minutes so the admin UI's usage graphs have history to plot, and once
// immediately on startup so a freshly deployed instance isn't empty.
// runMetricsSampler's snapshot is only ever this instance's own
// in-process counters (metrics.CurrentTotals reads local atomics), so in a
// multi-instance deployment the recorded row reflects whichever instance
// currently holds the lease, not the cluster's combined traffic. The lease
// still does its job of preventing duplicate rows; it doesn't make the
// numbers cluster-wide -- that would need shared counters (e.g. Redis
// INCR) instead of in-process atomics, which is a separate change.
func runMetricsSampler(ctx context.Context, history *metrics.History, storageCfg config.StorageConfig, lease *leaderlease.Lease, logger *slog.Logger) {
	const interval = 10 * time.Minute
	record := func() {
		if leader, err := lease.TryAcquire(ctx, "metrics-sampler", interval*2); err != nil || !leader {
			return
		}
		var storageBytes int64
		if storageCfg.Backend == "local" {
			var err error
			storageBytes, err = storage.LocalDirSize(storageCfg.LocalDir)
			if err != nil {
				logger.Warn("storage usage measurement failed", "error", err)
			}
		}
		if err := history.Record(ctx, metrics.CurrentTotals(), storageBytes); err != nil {
			logger.Warn("metrics snapshot failed", "error", err)
		}
	}
	record()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			record()
		}
	}
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
