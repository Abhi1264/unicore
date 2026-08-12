package handlers

import (
	"encoding/json"

	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

type TenantsHandler struct {
	admin *services.AdminService
	pool  *db.Pool
}

func NewTenantsHandler(admin *services.AdminService, pool *db.Pool) *TenantsHandler {
	return &TenantsHandler{admin: admin, pool: pool}
}

func (h *TenantsHandler) GetCurrent(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	// `tenants` is a platform table with no tenant_id column to scope by. The id
	// used here comes from host resolution, not from the request body, so this
	// cannot be steered at another institute's row.
	tenant, err := h.pool.Platform().GetTenantByID(c.Context(), tenantID)
	if err != nil {
		return mapSvcError(c, err)
	}
	cfg, err := h.admin.GetTenantConfig(c.Context(), tenantID)
	if err != nil {
		return mapSvcError(c, err)
	}
	var branding map[string]any
	_ = json.Unmarshal(cfg.Branding, &branding)
	return JSON(c, fiber.StatusOK, fiber.Map{
		"tenant": tenant,
		"branding": branding,
		"config": fiber.Map{
			"grading_system":            cfg.GradingSystem,
			"academic_calendar_type":    cfg.AcademicCalendarType,
			"attendance_threshold_pct":  cfg.AttendanceThresholdPct,
			"grading_scale":             json.RawMessage(cfg.GradingScale),
		},
	})
}

func (h *TenantsHandler) List(c *fiber.Ctx) error {
	status := c.Query("status", "")
	var list []sqlcdb.Tenant
	var err error
	if status == "pending_approval" || status == "pending" {
		list, err = h.admin.ListPendingTenants(c.Context())
	} else {
		list, err = h.admin.ListTenants(c.Context())
	}
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"tenants": list})
}

func (h *TenantsHandler) Approve(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid tenant id")
	}
	t, err := h.admin.ApproveTenant(c.Context(), id)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, t)
}

func (h *TenantsHandler) Reject(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid tenant id")
	}
	t, err := h.admin.RejectTenant(c.Context(), id)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, t)
}

func (h *TenantsHandler) Suspend(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid tenant id")
	}
	t, err := h.admin.SuspendTenant(c.Context(), id)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, t)
}

func (h *TenantsHandler) Reactivate(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid tenant id")
	}
	t, err := h.admin.ReactivateTenant(c.Context(), id)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, t)
}

type brandingBody struct {
	Branding             json.RawMessage `json:"branding"`
	GradingSystem        *string         `json:"grading_system"`
	AcademicCalendarType *string         `json:"academic_calendar_type"`
	GradingScale         json.RawMessage `json:"grading_scale"`
	AttendanceThreshold  *float64        `json:"attendance_threshold_pct"`
}

func (h *TenantsHandler) UpdateBranding(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	var body brandingBody
	if err := c.BodyParser(&body); err != nil {
		return JSONError(c, fiber.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}
	cfg, err := h.admin.UpdateBranding(c.Context(), tenantID, services.UpdateBrandingInput{
		GradingSystem:          body.GradingSystem,
		AcademicCalendarType:   body.AcademicCalendarType,
		Branding:               body.Branding,
		GradingScale:           body.GradingScale,
		AttendanceThresholdPct: body.AttendanceThreshold,
	})
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, cfg)
}
