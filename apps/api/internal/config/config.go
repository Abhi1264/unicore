package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Development defaults (also listed in insecureDefaults; keep both in sync).
const (
	DevJWTAccessSecret  = "dev-only-access-secret-not-for-production-0001"
	DevJWTRefreshSecret = "dev-only-refresh-secret-not-for-production-0002"
	DevWebhookSecret    = "dev-only-payment-webhook-secret-not-for-production"
	DevMetricsToken     = "dev-only-metrics-token-not-for-production"
	DevAppDBPassword    = "dev-only-app-db-password"
)

// insecureDefaults are public placeholder secrets; production must not use them.
var insecureDefaults = map[string]struct{}{
	"":                   {},
	DevJWTAccessSecret:   {},
	DevJWTRefreshSecret:  {},
	DevWebhookSecret:     {},
	DevMetricsToken:      {},
	DevAppDBPassword:     {},
	"dev-access-secret":  {},
	"dev-refresh-secret": {},
	"dev-access-secret-change-me-in-production":  {},
	"dev-refresh-secret-change-me-in-production": {},
	"changeme": {},
	"secret":   {},
	"password": {},
}

func isInsecureDefault(s string) bool {
	_, bad := insecureDefaults[s]
	return bad
}

const minSecretLength = 32

type Config struct {
	AppEnv               string
	BaseDomain           string
	APIAddr              string
	PlatformHost         string
	WebURL               string
	CORSOrigins          string
	DatabaseURL          string
	MigrateDatabaseURL   string
	RedisURL             string
	NATSURL              string
	JWTAccessSecret      string
	JWTRefreshSecret     string
	JWTAccessTTL         time.Duration
	JWTRefreshTTL        time.Duration
	StoragePath          string
	RateLimitIP          int
	RateLimitUser        int
	RateLimitWrite       int
	RateLimitWindow      time.Duration
	RateLimitAuthIP      int
	RateLimitAuthWindow  time.Duration
	PaymentWebhookSecret string
	MetricsToken         string
	MaxUploadBytes       int64
	VAPIDPublicKey       string
	VAPIDPrivateKey      string
	VAPIDSubject         string
	MigrationsPath       string
	AllowRLSBypass       bool
}

func Load() (*Config, error) {
	accessTTL, err := time.ParseDuration(getenv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("JWT_ACCESS_TTL: %w", err)
	}
	refreshTTL, err := time.ParseDuration(getenv("JWT_REFRESH_TTL", "168h"))
	if err != nil {
		return nil, fmt.Errorf("JWT_REFRESH_TTL: %w", err)
	}
	rlWindow, err := time.ParseDuration(getenv("RATE_LIMIT_WINDOW", "1m"))
	if err != nil {
		return nil, fmt.Errorf("RATE_LIMIT_WINDOW: %w", err)
	}
	authWindow, err := time.ParseDuration(getenv("RATE_LIMIT_AUTH_WINDOW", "15m"))
	if err != nil {
		return nil, fmt.Errorf("RATE_LIMIT_AUTH_WINDOW: %w", err)
	}

	cfg := &Config{
		AppEnv:               getenv("APP_ENV", "development"),
		BaseDomain:           getenv("APP_BASE_DOMAIN", "localhost"),
		APIAddr:              getenv("API_ADDR", ":8080"),
		PlatformHost:         getenv("PLATFORM_HOST", "app.localhost"),
		WebURL:               getenv("WEB_URL", "http://localhost:3000"),
		CORSOrigins:          getenv("CORS_ORIGINS", ""),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		MigrateDatabaseURL:   os.Getenv("DATABASE_MIGRATE_URL"),
		RedisURL:             getenv("REDIS_URL", "redis://localhost:6379/0"),
		NATSURL:              getenv("NATS_URL", "nats://localhost:4222"),
		JWTAccessSecret:      getenv("JWT_ACCESS_SECRET", DevJWTAccessSecret),
		JWTRefreshSecret:     getenv("JWT_REFRESH_SECRET", DevJWTRefreshSecret),
		JWTAccessTTL:         accessTTL,
		JWTRefreshTTL:        refreshTTL,
		StoragePath:          getenv("STORAGE_PATH", "./storage"),
		RateLimitIP:          getenvInt("RATE_LIMIT_IP", 100),
		RateLimitUser:        getenvInt("RATE_LIMIT_USER", 60),
		RateLimitWrite:       getenvInt("RATE_LIMIT_WRITE", 10),
		RateLimitWindow:      rlWindow,
		RateLimitAuthIP:      getenvInt("RATE_LIMIT_AUTH_IP", 10),
		RateLimitAuthWindow:  authWindow,
		PaymentWebhookSecret: os.Getenv("PAYMENT_WEBHOOK_SECRET"),
		MetricsToken:         os.Getenv("METRICS_TOKEN"),
		MaxUploadBytes:       int64(getenvInt("MAX_UPLOAD_BYTES", 10<<20)),
		VAPIDPublicKey:       os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey:      os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDSubject:         getenv("VAPID_SUBJECT", "mailto:admin@unicore.local"),
		MigrationsPath:       getenv("MIGRATIONS_PATH", "internal/db/migrations"),
		AllowRLSBypass:       os.Getenv("ALLOW_RLS_BYPASS") == "true",
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.MigrateDatabaseURL == "" {
		cfg.MigrateDatabaseURL = cfg.DatabaseURL
	}
	if err := cfg.validateSecrets(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

// validateSecrets refuses placeholder or short signing keys.
func (c *Config) validateSecrets() error {
	const generate = "generate one with: openssl rand -base64 48"

	for name, secret := range map[string]string{
		"JWT_ACCESS_SECRET":  c.JWTAccessSecret,
		"JWT_REFRESH_SECRET": c.JWTRefreshSecret,
	} {
		if len(secret) < minSecretLength {
			return fmt.Errorf("%s must be at least %d characters (%s)", name, minSecretLength, generate)
		}
		// Dev defaults pass the length check; reject them by name in production.
		if c.IsProduction() && isInsecureDefault(secret) {
			return fmt.Errorf("%s is still set to the public development default (%s)", name, generate)
		}
	}
	if c.JWTAccessSecret == c.JWTRefreshSecret {
		return fmt.Errorf("JWT_ACCESS_SECRET and JWT_REFRESH_SECRET must differ so a refresh token cannot be replayed as an access token")
	}

	if !c.IsProduction() {
		return nil
	}

	if c.AllowRLSBypass {
		return fmt.Errorf("ALLOW_RLS_BYPASS must not be enabled in production; it disables tenant isolation")
	}
	if c.PaymentWebhookSecret == "" || isInsecureDefault(c.PaymentWebhookSecret) {
		return fmt.Errorf("PAYMENT_WEBHOOK_SECRET must be set to a unique value in production; the payment confirmation endpoint is unauthenticated without it (%s)", generate)
	}
	if c.MetricsToken == "" || isInsecureDefault(c.MetricsToken) {
		return fmt.Errorf("METRICS_TOKEN must be set to a unique value in production; /metrics exposes internal request and queue data (%s)", generate)
	}
	if strings.Contains(c.DatabaseURL, "sslmode=disable") {
		return fmt.Errorf("DATABASE_URL must not use sslmode=disable in production")
	}
	if !strings.HasPrefix(c.WebURL, "https://") {
		return fmt.Errorf("WEB_URL must use https in production, got %q", c.WebURL)
	}
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
