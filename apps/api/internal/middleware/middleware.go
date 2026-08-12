package middleware

import (
	"log/slog"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/cache"
	"github.com/Abhi1264/unicore/api/internal/config"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/Abhi1264/unicore/api/internal/metrics"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type CtxKey string

const (
	KeyRequestID    = "request_id"
	KeyTenant       = "tenant"
	KeyClaims       = "claims"
	KeyPlatformHost = "platform_host"
)

type TenantInfo struct {
	ID     uuid.UUID
	Slug   string
	Name   string
	Status string
}

func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Client-supplied ids are attacker-controlled and logged; cap and validate.
		id := c.Get("X-Request-ID")
		if !validRequestID(id) {
			id = uuid.NewString()
		}
		c.Locals(KeyRequestID, id)
		c.Set("X-Request-ID", id)
		return c.Next()
	}
}

func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func AccessLog(log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		path := c.Route().Path
		if path == "" {
			path = c.Path()
		}
		status := c.Response().StatusCode()
		elapsed := time.Since(start)
		metrics.HTTPRequestDuration.WithLabelValues(c.Method(), path, statusLabel(status)).Observe(elapsed.Seconds())

		// Log the route template, never c.Path() (raw paths carry ids/query strings).
		attrs := []any{
			"request_id", c.Locals(KeyRequestID),
			"method", c.Method(),
			"route", path,
			"status", status,
			"latency_ms", elapsed.Milliseconds(),
		}
		if t, ok := TenantFromCtx(c); ok {
			attrs = append(attrs, "tenant", t.Slug)
		}
		log.Info("request", attrs...)
		return err
	}
}

func statusLabel(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	default:
		return "2xx"
	}
}

// isPlatformHost reports whether the request arrived on the control-plane host.
func isPlatformHost(cfg *config.Config, host string) bool {
	if host == cfg.PlatformHost {
		return true
	}
	if cfg.IsProduction() {
		return false
	}
	return host == "localhost" || host == "127.0.0.1" || host == cfg.BaseDomain
}

func ResolveTenant(cfg *config.Config, pool *db.Pool, log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch c.Path() {
		case "/healthz", "/readyz", "/metrics":
			return c.Next()
		}

		host := strings.ToLower(c.Hostname())
		host = strings.Split(host, ":")[0]

		ctx := c.Context()
		q := pool.Platform()

		if isPlatformHost(cfg, host) {
			c.Locals(KeyPlatformHost, true)
			// X-Tenant-Slug only selects a tenant; Authenticate still binds the token.
			if slug := c.Get("X-Tenant-Slug"); slug != "" {
				if !validSlug(slug) {
					return fiber.NewError(fiber.StatusBadRequest, "invalid tenant slug")
				}
				t, err := q.GetTenantBySlug(ctx, slug)
				if err != nil {
					return fiber.NewError(fiber.StatusNotFound, "tenant not found")
				}
				if t.Status == "suspended" || t.Status == "rejected" {
					return fiber.NewError(fiber.StatusForbidden, "tenant suspended")
				}
				c.Locals(KeyTenant, TenantInfo{ID: t.ID, Slug: t.Slug, Name: t.Name, Status: string(t.Status)})
			}
			return c.Next()
		}

		var t sqlcdb.Tenant
		var err error
		if suffix := "." + cfg.BaseDomain; strings.HasSuffix(host, suffix) {
			slug := strings.TrimSuffix(host, suffix)
			if !validSlug(slug) {
				return fiber.NewError(fiber.StatusNotFound, "unknown tenant host")
			}
			t, err = q.GetTenantBySlug(ctx, slug)
		} else {
			t, err = q.GetTenantByCustomDomain(ctx, pgtype.Text{String: host, Valid: true})
		}
		if err != nil {
			log.Warn("tenant resolve failed", "host", host)
			return fiber.NewError(fiber.StatusNotFound, "unknown tenant host")
		}
		if t.Status == "suspended" || t.Status == "rejected" {
			return fiber.NewError(fiber.StatusForbidden, "tenant suspended")
		}
		c.Locals(KeyTenant, TenantInfo{ID: t.ID, Slug: t.Slug, Name: t.Name, Status: string(t.Status)})
		return c.Next()
	}
}

// validSlug mirrors tenants_slug_format so a hostile Host never reaches the DB.
func validSlug(s string) bool {
	if len(s) < 3 || len(s) > 63 {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r == '-' && i > 0 && i < len(s)-1:
		default:
			return false
		}
	}
	return true
}

// Authenticate validates the bearer token and binds it to the resolved tenant.
func Authenticate(tokens *auth.TokenService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		h := c.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
		}
		claims, err := tokens.ParseAccess(strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
		}

		platform, _ := c.Locals(KeyPlatformHost).(bool)
		ti, hasTenant := TenantFromCtx(c)

		switch {
		case claims.Role == auth.RoleSuperadmin:
			// Superadmin tokens must stay on the control plane.
			if !platform {
				return fiber.NewError(fiber.StatusForbidden, "superadmin must use the platform host")
			}
		case !hasTenant:
			return fiber.NewError(fiber.StatusBadRequest, "tenant host required")
		case claims.TenantID != ti.ID:
			return fiber.NewError(fiber.StatusForbidden, "token tenant mismatch")
		}

		c.Locals(KeyClaims, claims)
		return c.Next()
	}
}

// RequirePlatformHost keeps control-plane routes off institute subdomains.
func RequirePlatformHost() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if platform, _ := c.Locals(KeyPlatformHost).(bool); !platform {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		return c.Next()
	}
}

func RequireRoles(roles ...auth.Role) fiber.Handler {
	set := make(map[auth.Role]struct{}, len(roles))
	for _, r := range roles {
		set[r] = struct{}{}
	}
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(KeyClaims).(*auth.Claims)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		if _, ok := set[claims.Role]; !ok {
			return fiber.NewError(fiber.StatusForbidden, "insufficient role")
		}
		return c.Next()
	}
}

// RateLimitIP applies a fixed-window counter per source IP (shared via Redis).
func RateLimitIP(cfg *config.Config, redis *cache.Client, log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if redis == nil || !redis.Available() {
			// Fail open for availability; make the outage visible.
			log.Warn("rate limiting disabled: redis unavailable", "path", c.Path())
			return c.Next()
		}
		if err := enforce(c, redis, "rl:ip:"+clientIP(c), cfg.RateLimitIP, cfg.RateLimitWindow); err != nil {
			return err
		}
		return c.Next()
	}
}

// RateLimitUser budgets requests per authenticated user; mount after Authenticate.
func RateLimitUser(cfg *config.Config, redis *cache.Client) fiber.Handler {
	return userScoped(redis, "rl:user:", func() int { return cfg.RateLimitUser }, func() time.Duration { return cfg.RateLimitWindow })
}

// RateLimitWrite is a tight per-user budget for payments, enrolments, and bulk imports.
func RateLimitWrite(cfg *config.Config, redis *cache.Client) fiber.Handler {
	return userScoped(redis, "rl:write:", func() int { return cfg.RateLimitWrite }, func() time.Duration { return cfg.RateLimitWindow })
}

func userScoped(redis *cache.Client, prefix string, limit func() int, window func() time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if redis == nil || !redis.Available() {
			return c.Next()
		}
		claims, ok := c.Locals(KeyClaims).(*auth.Claims)
		if !ok {
			return c.Next()
		}
		key := prefix + claims.UserID.String()
		if prefix == "rl:write:" {
			key += ":" + c.Route().Path
		}
		if err := enforce(c, redis, key, limit(), window()); err != nil {
			return err
		}
		return c.Next()
	}
}

// RateLimitAuth is a tight per-IP budget for credential endpoints.
// Unlike other limiters, it falls back to in-process counters when Redis is down.
func RateLimitAuth(cfg *config.Config, redis *cache.Client, log *slog.Logger) fiber.Handler {
	fallback := newLocalLimiter()
	return func(c *fiber.Ctx) error {
		scope := c.Route().Path + ":" + clientIP(c)
		if redis == nil || !redis.Available() {
			log.Warn("auth rate limiting degraded to in-process counters", "route", c.Route().Path)
			if retry, ok := fallback.exceeded(scope, cfg.RateLimitAuthIP, cfg.RateLimitAuthWindow); ok {
				setRateLimitHeaders(c, cfg.RateLimitAuthIP, 0, retry)
				return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded")
			}
			return c.Next()
		}
		if err := enforce(c, redis, "rl:auth:"+scope, cfg.RateLimitAuthIP, cfg.RateLimitAuthWindow); err != nil {
			return err
		}
		return c.Next()
	}
}

// enforce returns 429 when the window budget is exhausted; it never advances the chain.
func enforce(c *fiber.Ctx, redis *cache.Client, key string, limit int, window time.Duration) error {
	n, err := redis.Incr(c.Context(), key)
	if err != nil {
		return nil
	}
	if n == 1 {
		_ = redis.Expire(c.Context(), key, window)
	}
	if int(n) > limit {
		retry := window
		if ttl, err := redis.TTL(c.Context(), key); err == nil && ttl > 0 {
			retry = ttl
		}
		setRateLimitHeaders(c, limit, 0, retry)
		return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded")
	}
	setRateLimitHeaders(c, limit, limit-int(n), 0)
	return nil
}

func setRateLimitHeaders(c *fiber.Ctx, limit, remaining int, retry time.Duration) {
	c.Set("X-RateLimit-Limit", strconv.Itoa(limit))
	c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	if retry > 0 {
		// Retry-After is whole seconds; ceil so we don't invite retries still in-window.
		c.Set("Retry-After", strconv.Itoa(int(math.Ceil(retry.Seconds()))))
	}
}

// localLimiter is the in-process fallback for credential endpoints when Redis is down.
type localLimiter struct {
	mu      sync.Mutex
	windows map[string]*localWindow
}

type localWindow struct {
	count   int
	resetAt time.Time
}

func newLocalLimiter() *localLimiter {
	return &localLimiter{windows: make(map[string]*localWindow)}
}

const localLimiterMaxKeys = 10000

func (l *localLimiter) exceeded(key string, limit int, window time.Duration) (time.Duration, bool) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	w, ok := l.windows[key]
	if !ok || now.After(w.resetAt) {
		if len(l.windows) >= localLimiterMaxKeys {
			for k, v := range l.windows {
				if now.After(v.resetAt) {
					delete(l.windows, k)
				}
			}
			// Cap reached with no expired keys: refuse new keys (safe under spray).
			if len(l.windows) >= localLimiterMaxKeys {
				return window, true
			}
		}
		w = &localWindow{resetAt: now.Add(window)}
		l.windows[key] = w
	}
	w.count++
	if w.count > limit {
		return time.Until(w.resetAt), true
	}
	return 0, false
}

// clientIP trusts Fiber's ProxyHeader handling (only when trusted proxies are set).
func clientIP(c *fiber.Ctx) string {
	return c.IP()
}

func TenantFromCtx(c *fiber.Ctx) (TenantInfo, bool) {
	t, ok := c.Locals(KeyTenant).(TenantInfo)
	return t, ok
}

func ClaimsFromCtx(c *fiber.Ctx) (*auth.Claims, bool) {
	cl, ok := c.Locals(KeyClaims).(*auth.Claims)
	return cl, ok
}
