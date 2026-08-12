package handlers

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

type DocumentsHandler struct {
	svc         *services.DocumentsService
	pool        *db.Pool
	storagePath string
}

func NewDocumentsHandler(svc *services.DocumentsService, pool *db.Pool, storagePath string) *DocumentsHandler {
	abs, err := filepath.Abs(storagePath)
	if err != nil {
		abs = storagePath
	}
	return &DocumentsHandler{svc: svc, pool: pool, storagePath: abs}
}

func (h *DocumentsHandler) Request(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}
	var body struct {
		Type string `json:"type"`
	}
	if err := parseBody(c, &body); err != nil {
		return err
	}
	sid, err := studentIDForUser(c.Context(), h.pool, tenantID, claims.UserID)
	if err != nil {
		return mapSvcError(c, err)
	}
	doc, err := h.svc.RequestDocument(c.Context(), tenantID, sid, body.Type)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusCreated, doc)
}

func (h *DocumentsHandler) List(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}
	studentID, err := resolveStudentScope(c, h.pool, tenantID, claims)
	if err != nil {
		return err
	}
	list, err := h.svc.ListDocuments(c.Context(), tenantID, studentID)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"documents": list})
}

func (h *DocumentsHandler) Download(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid document id")
	}
	doc, err := h.svc.GetDocument(c.Context(), tenantID, id)
	if err != nil {
		return mapSvcError(c, err)
	}

	// Students may only download their own documents (tenant scope is not enough).
	if claims.Role == auth.RoleStudent {
		sid, err := studentIDForUser(c.Context(), h.pool, tenantID, claims.UserID)
		if err != nil {
			return mapSvcError(c, err)
		}
		if doc.StudentID != sid {
			return JSONError(c, fiber.StatusNotFound, "NOT_FOUND", "resource not found")
		}
	}

	if doc.Status != "ready" || !doc.StorageRef.Valid || doc.StorageRef.String == "" {
		return JSONError(c, fiber.StatusConflict, "NOT_READY", "document not ready")
	}

	path, ok := h.resolveStoragePath(doc.StorageRef.String)
	if !ok {
		logInternal(c, os.ErrInvalid)
		return JSONError(c, fiber.StatusNotFound, "NOT_FOUND", "resource not found")
	}
	if _, err := os.Stat(path); err != nil {
		return JSONError(c, fiber.StatusNotFound, "NOT_FOUND", "resource not found")
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", `attachment; filename="`+string(doc.Type)+`.pdf"`)
	c.Set("X-Content-Type-Options", "nosniff")
	return c.SendFile(path)
}

// resolveStoragePath confines storage_ref to the configured storage directory.
func (h *DocumentsHandler) resolveStoragePath(ref string) (string, bool) {
	if ref == "" || filepath.IsAbs(ref) || strings.ContainsRune(ref, 0) {
		return "", false
	}
	full := filepath.Join(h.storagePath, filepath.Clean("/"+ref))
	if full != h.storagePath && !strings.HasPrefix(full, h.storagePath+string(os.PathSeparator)) {
		return "", false
	}
	return full, true
}
