package main

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/cache"
	"github.com/Abhi1264/unicore/api/internal/config"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/handlers"
	"github.com/Abhi1264/unicore/api/internal/middleware"
	"github.com/Abhi1264/unicore/api/internal/queue"
	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/Abhi1264/unicore/api/internal/ws"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Migrate as schema owner; runtime pool uses the restricted role (see AssertRLSEnforced).
	if err := db.RunMigrations(cfg.MigrateDatabaseURL, cfg.MigrationsPath); err != nil {
		log.Error("migrations", "error", err)
		os.Exit(1)
	}

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.AssertRLSEnforced(ctx); err != nil {
		if !cfg.AllowRLSBypass {
			log.Error("refusing to start", "error", err)
			os.Exit(1)
		}
		log.Warn("starting with row-level security bypassed (ALLOW_RLS_BYPASS=true); tenant isolation is NOT enforced")
	}

	redis, err := cache.New(ctx, cfg.RedisURL, log)
	if err != nil {
		log.Error("redis", "error", err)
		os.Exit(1)
	}
	defer redis.Close()

	natsClient, err := queue.New(cfg.NATSURL, log)
	if err != nil {
		log.Error("nats", "error", err)
		os.Exit(1)
	}
	defer natsClient.Close()

	tokens := auth.NewTokenService(cfg.JWTAccessSecret, cfg.JWTRefreshSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	hub := ws.NewHub()

	if err := os.MkdirAll(cfg.StoragePath, 0o750); err != nil {
		log.Error("storage path", "error", err)
		os.Exit(1)
	}

	deps := handlers.Deps{
		Pool:          pool,
		Tokens:        tokens,
		Redis:         redis,
		Log:           log,
		StoragePath:   cfg.StoragePath,
		Hub:           hub,
		Auth:          services.NewAuthService(pool, redis, tokens),
		Admin:         services.NewAdminService(pool),
		Results:       services.NewResultsService(pool, redis),
		Academic:      services.NewAcademicService(pool),
		Fees:          services.NewFeesService(pool),
		Attendance:    services.NewAttendanceService(pool),
		Announcements: services.NewAnnouncementsService(pool),
		Documents:     services.NewDocumentsService(pool),
	}

	app := fiber.New(fiber.Config{
		AppName:      "unicore-api",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
		BodyLimit: int(cfg.MaxUploadBytes) + (1 << 20),
		// Only trust X-Forwarded-* from declared proxies (rate-limit IP integrity).
		EnableTrustedProxyCheck: true,
		ErrorHandler:            errorHandler(log),
	})

	app.Use(recover.New())
	app.Use(helmet.New(helmet.Config{
		XSSProtection:         "0",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		CrossOriginOpenerPolicy: "same-origin",
		HSTSMaxAge:            63072000,
		HSTSPreloadEnabled:    cfg.IsProduction(),
	}))
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: corsOriginAllowed(cfg),
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Request-ID, X-Tenant-Slug, Idempotency-Key, X-Signature",
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		ExposeHeaders:    "X-Request-ID, Retry-After, X-RateLimit-Limit, X-RateLimit-Remaining",
		AllowCredentials: true,
		MaxAge:           600,
	}))
	app.Use(middleware.RequestID())
	app.Use(middleware.AccessLog(log))
	app.Use(middleware.ResolveTenant(cfg, pool, log))
	app.Use(middleware.RateLimitIP(cfg, redis, log))

	handlers.Register(app, cfg, deps)

	go func() {
		log.Info("listening", "addr", cfg.APIAddr, "env", cfg.AppEnv)
		if err := app.Listen(cfg.APIAddr); err != nil {
			log.Error("listen", "error", err)
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Info("shutting down")
	_ = app.ShutdownWithTimeout(15 * time.Second)
}

// errorCodes maps middleware sentinel messages to stable API codes.
var errorCodes = map[string]string{
	"tenant host required":                  "TENANT_REQUIRED",
	"invalid tenant slug":                   "TENANT_NOT_FOUND",
	"tenant not found":                      "TENANT_NOT_FOUND",
	"unknown tenant host":                   "TENANT_NOT_FOUND",
	"tenant suspended":                      "TENANT_NOT_ACTIVE",
	"missing bearer token":                  "UNAUTHORIZED",
	"invalid token":                         "UNAUTHORIZED",
	"unauthorized":                          "UNAUTHORIZED",
	"token tenant mismatch":                 "FORBIDDEN",
	"insufficient role":                     "FORBIDDEN",
	"superadmin must use the platform host": "FORBIDDEN",
	"rate limit exceeded":                   "RATE_LIMITED",
	"not found":                             "NOT_FOUND",
}

func errorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		if errors.Is(err, handlers.ErrResponseWritten) {
			return nil
		}
		if e, ok := err.(*fiber.Error); ok {
			code, known := errorCodes[e.Message]
			if !known {
				if e.Code >= 500 {
					log.Error("server error", "request_id", c.Locals(middleware.KeyRequestID), "error", e.Message)
					return handlers.JSONError(c, e.Code, "INTERNAL", "internal error")
				}
				code = "VALIDATION_ERROR"
			}
			return handlers.JSONError(c, e.Code, code, e.Message)
		}
		log.Error("unhandled error",
			"request_id", c.Locals(middleware.KeyRequestID),
			"route", c.Route().Path,
			"error", err,
		)
		return handlers.JSONError(c, fiber.StatusInternalServerError, "INTERNAL", "internal error")
	}
}

// corsOriginAllowed permits WebURL, CORS_ORIGINS, and hosts under BaseDomain (never "*").
func corsOriginAllowed(cfg *config.Config) func(string) bool {
	allowList := map[string]struct{}{}
	if cfg.WebURL != "" {
		allowList[cfg.WebURL] = struct{}{}
	}
	for _, o := range strings.Split(cfg.CORSOrigins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			allowList[o] = struct{}{}
		}
	}
	production := cfg.IsProduction()
	base := strings.ToLower(cfg.BaseDomain)

	return func(origin string) bool {
		if origin == "" {
			return false
		}
		if _, ok := allowList[origin]; ok {
			return true
		}
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return false
		}
		host := strings.ToLower(u.Hostname())
		if !production && (host == "localhost" || host == "127.0.0.1" || strings.HasSuffix(host, ".localhost")) {
			return true
		}
		if production && u.Scheme != "https" {
			return false
		}
		return base != "" && (host == base || strings.HasSuffix(host, "."+base))
	}
}
