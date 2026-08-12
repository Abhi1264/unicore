package handlers

import (
	"time"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type FeesHandler struct {
	svc           *services.FeesService
	pool          *db.Pool
	webhookSecret string
}

func NewFeesHandler(svc *services.FeesService, pool *db.Pool, webhookSecret string) *FeesHandler {
	return &FeesHandler{svc: svc, pool: pool, webhookSecret: webhookSecret}
}

func (h *FeesHandler) ListHeads(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	list, err := h.svc.ListFeeHeads(c.Context(), tenantID)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"fee_heads": list})
}

func (h *FeesHandler) CreateHead(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	var body struct {
		Name               string   `json:"name"`
		Amount             float64  `json:"amount"`
		DueDate            *string  `json:"due_date"`
		LateFeeAmount      float64  `json:"late_fee_amount"`
		ApplicablePrograms []string `json:"applicable_programs"`
	}
	if err := parseBody(c, &body); err != nil {
		return err
	}
	if err := requireText("name", body.Name, 200); err != nil {
		return err
	}
	if err := requireAmount("amount", body.Amount); err != nil {
		return err
	}
	if err := requireAmount("late_fee_amount", body.LateFeeAmount); err != nil {
		return err
	}
	if len(body.ApplicablePrograms) > 100 {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "too many applicable_programs")
	}
	var due *time.Time
	if body.DueDate != nil && *body.DueDate != "" {
		t, err := time.Parse("2006-01-02", *body.DueDate)
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "due_date must be YYYY-MM-DD")
		}
		due = &t
	}
	fh, err := h.svc.CreateFeeHead(c.Context(), tenantID, services.CreateFeeHeadInput{
		Name: body.Name, Amount: body.Amount, DueDate: due,
		LateFeeAmount: body.LateFeeAmount, ApplicablePrograms: body.ApplicablePrograms,
	})
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusCreated, fh)
}

func (h *FeesHandler) ListDues(c *fiber.Ctx) error {
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
	list, err := h.svc.ListDues(c.Context(), tenantID, studentID)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"dues": list})
}

func (h *FeesHandler) Pay(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}
	key, err := requireIdempotencyKey(c)
	if err != nil {
		return err
	}
	var body struct {
		FeeHeadID uuid.UUID `json:"fee_head_id"`
	}
	if err := parseBody(c, &body); err != nil {
		return err
	}
	if body.FeeHeadID == uuid.Nil {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "fee_head_id is required")
	}
	// Payer is always the authenticated student; amount is derived server-side.
	sid, err := studentIDForUser(c.Context(), h.pool, tenantID, claims.UserID)
	if err != nil {
		return mapSvcError(c, err)
	}
	pmt, err := h.svc.CreatePayment(c.Context(), tenantID, services.CreatePaymentInput{
		StudentID: sid, FeeHeadID: body.FeeHeadID, IdempotencyKey: key,
	})
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusCreated, pmt)
}

// ConfirmWebhook settles a payment via HMAC-SHA256 over the body (PAYMENT_WEBHOOK_SECRET).
func (h *FeesHandler) ConfirmWebhook(c *fiber.Ctx) error {
	if h.webhookSecret == "" {
		return JSONError(c, fiber.StatusServiceUnavailable, "WEBHOOK_NOT_CONFIGURED",
			"payment confirmation is disabled until PAYMENT_WEBHOOK_SECRET is configured")
	}
	if !auth.VerifyHMACSignature(h.webhookSecret, c.Body(), c.Get("X-Signature")) {
		return JSONError(c, fiber.StatusUnauthorized, "INVALID_SIGNATURE", "invalid webhook signature")
	}

	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	var body struct {
		PaymentID  uuid.UUID `json:"payment_id"`
		GatewayRef string    `json:"gateway_ref"`
	}
	if err := parseBody(c, &body); err != nil {
		return err
	}
	if body.PaymentID == uuid.Nil {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "payment_id is required")
	}
	if len(body.GatewayRef) > 200 {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "gateway_ref too long")
	}
	pmt, err := h.svc.ConfirmPayment(c.Context(), tenantID, body.PaymentID, body.GatewayRef)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, pmt)
}
