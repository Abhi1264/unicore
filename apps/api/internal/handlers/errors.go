package handlers

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/Abhi1264/unicore/api/internal/middleware"
	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// mapSvcError maps known domain errors; unknowns become a generic 500 (no driver leaks).
func mapSvcError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, services.ErrNotFound), errors.Is(err, pgx.ErrNoRows):
		return JSONError(c, fiber.StatusNotFound, "NOT_FOUND", "resource not found")
	case errors.Is(err, services.ErrUnauthorized), errors.Is(err, services.ErrInvalidCredentials):
		return JSONError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
	case errors.Is(err, services.ErrTenantNotActive):
		return JSONError(c, fiber.StatusForbidden, "TENANT_NOT_ACTIVE", "this institute is not active")
	case errors.Is(err, services.ErrForbidden):
		return JSONError(c, fiber.StatusForbidden, "FORBIDDEN", "you do not have access to this resource")
	case errors.Is(err, services.ErrSeatFull):
		return JSONError(c, fiber.StatusConflict, "SEATS_FULL", services.ErrSeatFull.Error())
	case errors.Is(err, services.ErrRegistrationClosed):
		return JSONError(c, fiber.StatusConflict, "REGISTRATION_CLOSED", services.ErrRegistrationClosed.Error())
	case errors.Is(err, services.ErrConflict):
		return JSONError(c, fiber.StatusConflict, "CONFLICT", safeMessage(err, "this action conflicts with existing data"))
	case errors.Is(err, services.ErrInvalidInput):
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", safeMessage(err, "invalid input"))
	default:
		logInternal(c, err)
		return JSONError(c, fiber.StatusInternalServerError, "INTERNAL", "internal error")
	}
}

// safeMessage forwards an operator-authored suffix; strips driver/SQL fragments.
func safeMessage(err error, fallback string) string {
	msg := err.Error()
	if i := strings.LastIndex(msg, ": "); i >= 0 {
		msg = msg[i+2:]
	}
	lower := strings.ToLower(msg)
	switch {
	case msg == "", len(msg) > 200,
		strings.Contains(msg, "SQLSTATE"),
		strings.Contains(lower, "pq:"),
		strings.Contains(lower, "error:"),
		strings.Contains(msg, "/"),
		strings.Contains(msg, "\\"):
		return fallback
	default:
		return msg
	}
}

func logInternal(c *fiber.Ctx, err error) {
	slog.Default().Error("request failed",
		"request_id", c.Locals(middleware.KeyRequestID),
		"route", c.Route().Path,
		"error", err,
	)
}

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(name))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, errors.New("invalid id")
	}
	return id, nil
}

func requireTenantID(c *fiber.Ctx) (uuid.UUID, error) {
	ti, ok := middleware.TenantFromCtx(c)
	if !ok {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "tenant host required")
	}
	return ti.ID, nil
}

func requireClaims(c *fiber.Ctx) (*auth.Claims, error) {
	claims, ok := middleware.ClaimsFromCtx(c)
	if !ok {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	return claims, nil
}

func parseBody(c *fiber.Ctx, dst any) error {
	if err := c.BodyParser(dst); err != nil {
		return JSONError(c, fiber.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}
	return nil
}

const maxIdempotencyKeyLength = 128

func requireIdempotencyKey(c *fiber.Ctx) (string, error) {
	key := strings.TrimSpace(c.Get("Idempotency-Key"))
	if key == "" {
		return "", JSONError(c, fiber.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key header required")
	}
	if len(key) > maxIdempotencyKeyLength {
		return "", JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "Idempotency-Key too long")
	}
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == ':':
		default:
			return "", JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "Idempotency-Key contains unsupported characters")
		}
	}
	return key, nil
}

func requireText(field, v string, max int) error {
	v = strings.TrimSpace(v)
	if v == "" {
		return fiber.NewError(fiber.StatusBadRequest, field+" is required")
	}
	if utf8.RuneCountInString(v) > max {
		return fiber.NewError(fiber.StatusBadRequest, field+" is too long")
	}
	return nil
}

func requireAmount(field string, v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1e9 {
		return fiber.NewError(fiber.StatusBadRequest, field+" must be between 0 and 1000000000")
	}
	return nil
}

func requireSemester(v string) error {
	return requireText("semester", v, 32)
}

func studentIDForUser(ctx context.Context, pool *db.Pool, tenantID, userID uuid.UUID) (uuid.UUID, error) {
	var sid uuid.UUID
	err := pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		st, err := q.GetStudentByUserID(ctx, sqlcdb.GetStudentByUserIDParams{
			TenantID: tenantID,
			UserID:   userID,
		})
		if err != nil {
			return err
		}
		sid = st.ID
		return nil
	})
	return sid, err
}

// resolveStudentScope pins students to their own record; staff must name student_id explicitly.
func resolveStudentScope(c *fiber.Ctx, pool *db.Pool, tenantID uuid.UUID, claims *auth.Claims) (uuid.UUID, error) {
	if claims.Role == auth.RoleStudent {
		sid, err := studentIDForUser(c.Context(), pool, tenantID, claims.UserID)
		if err != nil {
			return uuid.Nil, mapSvcError(c, err)
		}
		return sid, nil
	}

	raw := c.Query("student_id")
	if raw == "" {
		return uuid.Nil, JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "student_id query parameter is required")
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, JSONError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid student_id")
	}
	// Fail cross-tenant ids as 404 before downstream queries treat them as empty.
	if err := assertStudentInTenant(c.Context(), pool, tenantID, id); err != nil {
		return uuid.Nil, mapSvcError(c, err)
	}
	return id, nil
}

func assertStudentInTenant(ctx context.Context, pool *db.Pool, tenantID, studentID uuid.UUID) error {
	return pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		_, err := q.GetStudentByID(ctx, sqlcdb.GetStudentByIDParams{TenantID: tenantID, ID: studentID})
		if errors.Is(err, pgx.ErrNoRows) {
			return services.ErrNotFound
		}
		return err
	})
}

func assertCanTeach(c *fiber.Ctx, pool *db.Pool, tenantID, courseID uuid.UUID, semester string) error {
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}
	if claims.Role != auth.RoleFaculty {
		return nil
	}
	if err := services.NewAcademicService(pool).AssertFacultyTeaches(c.Context(), tenantID, claims.UserID, courseID, semester); err != nil {
		return mapSvcError(c, err)
	}
	return nil
}
