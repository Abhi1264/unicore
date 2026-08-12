package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

type AdminHandler struct {
	svc            *services.AdminService
	storagePath    string
	maxUploadBytes int64
}

func NewAdminHandler(svc *services.AdminService, storagePath string, maxUploadBytes int64) *AdminHandler {
	return &AdminHandler{svc: svc, storagePath: storagePath, maxUploadBytes: maxUploadBytes}
}

// bulkImportJobTypes is the closed set the worker accepts.
var bulkImportJobTypes = map[string]struct{}{
	"students": {},
	"results":  {},
}

func (h *AdminHandler) BulkImport(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}

	jobType := c.FormValue("job_type")
	if jobType == "" {
		jobType = c.Query("job_type", "students")
	}
	if _, ok := bulkImportJobTypes[jobType]; !ok {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "job_type must be one of: students, results")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "file is required")
	}
	if file.Size <= 0 || file.Size > h.maxUploadBytes {
		return JSONError(c, fiber.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "upload exceeds the maximum allowed size")
	}
	if !strings.EqualFold(filepath.Ext(file.Filename), ".csv") {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "only .csv uploads are accepted")
	}

	job, err := h.svc.CreateBulkImportJob(c.Context(), tenantID, jobType, claims.UserID)
	if err != nil {
		return mapSvcError(c, err)
	}

	// Path uses server-side uuids only; never the client filename.
	dir := filepath.Join(h.storagePath, "bulk", tenantID.String())
	if err := os.MkdirAll(dir, 0o750); err != nil {
		logInternal(c, err)
		return JSONError(c, fiber.StatusInternalServerError, "INTERNAL", "internal error")
	}
	dest := filepath.Join(dir, job.ID.String()+".csv")
	if err := c.SaveFile(file, dest); err != nil {
		logInternal(c, err)
		return JSONError(c, fiber.StatusInternalServerError, "INTERNAL", "failed to store upload")
	}

	return JSON(c, fiber.StatusAccepted, fiber.Map{"job": job})
}

func (h *AdminHandler) AuditLogs(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	limit := int32(c.QueryInt("limit", 100))
	if limit < 1 || limit > 500 {
		limit = 100
	}
	list, err := h.svc.ListAuditLogs(c.Context(), tenantID, limit)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"logs": list})
}

func (h *AdminHandler) Usage(c *fiber.Ctx) error {
	since := time.Now().AddDate(0, 0, -30)
	if q := c.Query("since"); q != "" {
		t, err := time.Parse("2006-01-02", q)
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "since must be YYYY-MM-DD")
		}
		since = t
	}
	list, err := h.svc.ListUsage(c.Context(), since)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"usage": list})
}
