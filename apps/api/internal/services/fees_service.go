package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/Abhi1264/unicore/api/internal/queue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type FeesService struct {
	pool *db.Pool
}

func NewFeesService(pool *db.Pool) *FeesService {
	return &FeesService{pool: pool}
}

type CreateFeeHeadInput struct {
	Name               string
	Amount             float64
	DueDate            *time.Time
	LateFeeAmount      float64
	ApplicablePrograms []string
}

type FeeDue struct {
	FeeHead       sqlcdb.FeeHead `json:"fee_head"`
	AmountDue     float64        `json:"amount_due"`
	LateFeeApplied bool          `json:"late_fee_applied"`
	IsOverdue     bool           `json:"is_overdue"`
}

func (s *FeesService) ListFeeHeads(ctx context.Context, tenantID uuid.UUID) ([]sqlcdb.FeeHead, error) {
	var out []sqlcdb.FeeHead
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.ListFeeHeads(ctx, tenantID)
		return err
	})
	return out, fmtErr("list fee heads", err)
}

func (s *FeesService) CreateFeeHead(ctx context.Context, tenantID uuid.UUID, in CreateFeeHeadInput) (sqlcdb.FeeHead, error) {
	if in.ApplicablePrograms == nil {
		in.ApplicablePrograms = []string{}
	}
	var due pgtype.Date
	if in.DueDate != nil {
		due = DateFromTime(*in.DueDate)
	}
	var out sqlcdb.FeeHead
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.CreateFeeHead(ctx, sqlcdb.CreateFeeHeadParams{
			TenantID:           tenantID,
			Name:               in.Name,
			Amount:             NumericFromFloat(in.Amount),
			DueDate:            due,
			LateFeeAmount:      NumericFromFloat(in.LateFeeAmount),
			ApplicablePrograms: in.ApplicablePrograms,
		})
		return err
	})
	return out, fmtErr("create fee head", err)
}

func (s *FeesService) ListDues(ctx context.Context, tenantID, studentID uuid.UUID) ([]FeeDue, error) {
	var dues []FeeDue
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		student, err := q.GetStudentByID(ctx, sqlcdb.GetStudentByIDParams{TenantID: tenantID, ID: studentID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}

		// sqlc's ListUnpaidFeeHeadsForStudent typing is wrong at the wire level; filter in Go.
		all, err := q.ListFeeHeads(ctx, tenantID)
		if err != nil {
			return err
		}
		payments, err := q.ListStudentPayments(ctx, sqlcdb.ListStudentPaymentsParams{
			TenantID:  tenantID,
			StudentID: studentID,
		})
		if err != nil {
			return err
		}
		paid := map[uuid.UUID]struct{}{}
		for _, p := range payments {
			if p.Status == "paid" {
				paid[p.FeeHeadID] = struct{}{}
			}
		}
		heads := make([]sqlcdb.FeeHead, 0, len(all))
		for _, h := range all {
			if _, ok := paid[h.ID]; ok {
				continue
			}
			if len(h.ApplicablePrograms) == 0 || containsString(h.ApplicablePrograms, student.Program) {
				heads = append(heads, h)
			}
		}

		now := time.Now().UTC()
		dues = make([]FeeDue, 0, len(heads))
		for _, h := range heads {
			amount, _ := FloatFromNumeric(h.Amount)
			late, _ := FloatFromNumeric(h.LateFeeAmount)
			overdue := h.DueDate.Valid && now.After(h.DueDate.Time.Add(24*time.Hour))
			lateApplied := overdue && late > 0
			dueAmt := amount
			if lateApplied {
				dueAmt += late
			}
			dues = append(dues, FeeDue{
				FeeHead:        h,
				AmountDue:      dueAmt,
				LateFeeApplied: lateApplied,
				IsOverdue:      overdue,
			})
		}
		return nil
	})
	return dues, fmtErr("list dues", err)
}

type CreatePaymentInput struct {
	StudentID      uuid.UUID
	FeeHeadID      uuid.UUID
	IdempotencyKey string
	GatewayRef     string
}

// errPaymentRaced signals a concurrent insert won; it never leaves CreatePayment.
var errPaymentRaced = errors.New("payment insert raced")

// CreatePayment is idempotent on (tenant_id, idempotency_key); unique constraint is authoritative.
func (s *FeesService) CreatePayment(ctx context.Context, tenantID uuid.UUID, in CreatePaymentInput) (sqlcdb.FeePayment, error) {
	if in.IdempotencyKey == "" {
		return sqlcdb.FeePayment{}, ErrInvalidInput
	}

	var payment sqlcdb.FeePayment
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		existing, err := q.GetFeePaymentByIdempotency(ctx, sqlcdb.GetFeePaymentByIdempotencyParams{
			TenantID:       tenantID,
			IdempotencyKey: in.IdempotencyKey,
		})
		if err == nil {
			return replayPayment(existing, in, &payment)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		head, err := q.GetFeeHead(ctx, sqlcdb.GetFeeHeadParams{TenantID: tenantID, ID: in.FeeHeadID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}

		amount, _ := FloatFromNumeric(head.Amount)
		late, _ := FloatFromNumeric(head.LateFeeAmount)
		now := time.Now().UTC()
		if head.DueDate.Valid && now.After(head.DueDate.Time.Add(24*time.Hour)) && late > 0 {
			amount += late
		}

		payment, err = q.CreateFeePayment(ctx, sqlcdb.CreateFeePaymentParams{
			TenantID:       tenantID,
			StudentID:      in.StudentID,
			FeeHeadID:      in.FeeHeadID,
			Amount:         NumericFromFloat(amount),
			Status:         "pending",
			IdempotencyKey: in.IdempotencyKey,
			GatewayRef:     TextOrEmpty(in.GatewayRef),
		})
		if isUniqueViolation(err) {
			// Must return: failed statement already aborted this transaction.
			return errPaymentRaced
		}
		if isForeignKeyViolation(err) {
			return ErrNotFound
		}
		return err
	})
	if !errors.Is(err, errPaymentRaced) {
		return payment, fmtErr("create payment", err)
	}

	// Re-read the winner in a fresh transaction (prior tx was rolled back).
	err = s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		existing, err := q.GetFeePaymentByIdempotency(ctx, sqlcdb.GetFeePaymentByIdempotencyParams{
			TenantID:       tenantID,
			IdempotencyKey: in.IdempotencyKey,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Collision on a different constraint (e.g. open payment for fee head).
				return ErrConflict
			}
			return err
		}
		return replayPayment(existing, in, &payment)
	})
	return payment, fmtErr("create payment", err)
}

// replayPayment returns a stored payment only when student/fee_head match (keys are per-tenant).
func replayPayment(existing sqlcdb.FeePayment, in CreatePaymentInput, out *sqlcdb.FeePayment) error {
	if existing.StudentID != in.StudentID || existing.FeeHeadID != in.FeeHeadID {
		return ErrConflict
	}
	*out = existing
	return nil
}

func (s *FeesService) ConfirmPayment(ctx context.Context, tenantID, paymentID uuid.UUID, gatewayRef string) (sqlcdb.FeePayment, error) {
	var payment sqlcdb.FeePayment
	err := s.pool.WithTenantTx(ctx, tenantID, func(ctx context.Context, _ pgx.Tx, q *sqlcdb.Queries) error {
		var err error
		payment, err = q.MarkFeePaymentPaid(ctx, sqlcdb.MarkFeePaymentPaidParams{
			TenantID:   tenantID,
			ID:         paymentID,
			GatewayRef: TextOrEmpty(gatewayRef),
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Already paid or missing — return existing if found.
				p, err2 := q.GetFeePaymentByID(ctx, sqlcdb.GetFeePaymentByIDParams{TenantID: tenantID, ID: paymentID})
				if err2 == nil {
					payment = p
					return nil
				}
				return ErrNotFound
			}
			return err
		}

		confirmPayload, _ := json.Marshal(map[string]any{
			"payment_id":  payment.ID,
			"student_id":  payment.StudentID,
			"fee_head_id": payment.FeeHeadID,
			"amount":      FloatFromNumericMust(payment.Amount),
			"tenant_id":   tenantID,
		})
		if _, err := q.InsertOutbox(ctx, sqlcdb.InsertOutboxParams{
			TenantID: tenantID,
			Topic:    queue.TopicPaymentConfirmed,
			Payload:  confirmPayload,
		}); err != nil {
			return err
		}

		docPayload, _ := json.Marshal(map[string]any{
			"type":        "fee_receipt",
			"payment_id":  payment.ID,
			"student_id":  payment.StudentID,
			"tenant_id":   tenantID,
		})
		_, err = q.InsertOutbox(ctx, sqlcdb.InsertOutboxParams{
			TenantID: tenantID,
			Topic:    queue.TopicPDFGenerate,
			Payload:  docPayload,
		})
		return err
	})
	return payment, fmtErr("confirm payment", err)
}

func FloatFromNumericMust(n pgtype.Numeric) float64 {
	f, _ := FloatFromNumeric(n)
	return f
}

func containsString(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}
