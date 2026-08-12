package handlers

import (
	"context"
	"time"

	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	svc  *services.AuthService
	pool *db.Pool
}

func NewAuthHandler(svc *services.AuthService, pool *db.Pool) *AuthHandler {
	return &AuthHandler{svc: svc, pool: pool}
}

type registerTenantBody struct {
	InstituteName string `json:"institute_name"`
	Subdomain     string `json:"subdomain"`
	AdminEmail    string `json:"admin_email"`
	AdminFullName string `json:"admin_full_name"`
	AdminPassword string `json:"admin_password"`
}

func (h *AuthHandler) RegisterTenant(c *fiber.Ctx) error {
	var body registerTenantBody
	if err := parseBody(c, &body); err != nil {
		return err
	}
	// Field-level validation lives in the service so the same rules apply to any
	// future caller (CLI, seed, admin console), not just this handler.
	res, err := h.svc.RegisterTenant(c.Context(), body.Subdomain, body.InstituteName, body.AdminEmail, body.AdminPassword, body.AdminFullName)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusCreated, fiber.Map{
		"tenant": res.Tenant,
		"admin":  publicUser(res.Admin),
	})
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	var body loginBody
	if err := parseBody(c, &body); err != nil {
		return err
	}
	res, err := h.svc.Login(c.Context(), tenantID, body.Email, body.Password, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{
		"user":   publicUser(res.User),
		"tokens": tokenPairJSON(res.Tokens),
	})
}

type refreshBody struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var body refreshBody
	if err := parseBody(c, &body); err != nil {
		return err
	}
	tokens, err := h.svc.Refresh(c.Context(), body.RefreshToken)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, tokenPairJSON(tokens))
}

// Logout revokes the presented refresh token. It always reports success so it
// cannot be used to probe which tokens are still live.
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var body refreshBody
	if err := parseBody(c, &body); err != nil {
		return err
	}
	_ = h.svc.Logout(c.Context(), body.RefreshToken)
	return JSON(c, fiber.StatusOK, fiber.Map{"status": "ok"})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}
	var user sqlcdb.User
	err = h.pool.WithTenant(c.Context(), claims.TenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var e error
		user, e = q.GetUserByID(ctx, sqlcdb.GetUserByIDParams{TenantID: claims.TenantID, ID: claims.UserID})
		return e
	})
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, publicUser(user))
}

// publicUser is the only shape a user is ever serialised in. Building it field
// by field (rather than marshalling sqlcdb.User) is what keeps password_hash
// out of every response that returns a user.
func publicUser(u sqlcdb.User) fiber.Map {
	return fiber.Map{
		"id":         u.ID,
		"tenant_id":  u.TenantID,
		"email":      u.Email,
		"full_name":  u.FullName,
		"role":       u.Role,
		"is_active":  u.IsActive,
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
	}
}

func tokenPairJSON(t services.TokenPair) fiber.Map {
	expiresIn := int64(time.Until(t.ExpiresAt).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}
	return fiber.Map{
		"access_token":  t.AccessToken,
		"refresh_token": t.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    expiresIn,
	}
}
