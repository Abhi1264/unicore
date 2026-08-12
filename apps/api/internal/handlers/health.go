package handlers

import (
	"strings"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/gofiber/fiber/v2"
)

// MetricsGuard requires a bearer token for /metrics (open only in development).
func MetricsGuard(token string, allowOpen bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if token == "" {
			if allowOpen {
				return c.Next()
			}
			return JSONError(c, fiber.StatusServiceUnavailable, "METRICS_DISABLED", "metrics require METRICS_TOKEN")
		}
		provided := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
		if !auth.SecretEqual(provided, token) {
			return JSONError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		}
		return c.Next()
	}
}

func Healthz(c *fiber.Ctx) error {
	return JSON(c, fiber.StatusOK, fiber.Map{"status": "ok"})
}

func Readyz(pool *db.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if pool == nil {
			return JSONError(c, fiber.StatusServiceUnavailable, "NOT_READY", "database unavailable")
		}
		if err := pool.Ping(c.Context()); err != nil {
			return JSONError(c, fiber.StatusServiceUnavailable, "NOT_READY", "database ping failed")
		}
		return JSON(c, fiber.StatusOK, fiber.Map{"status": "ready"})
	}
}
