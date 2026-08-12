package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Cross-tenant helpers via SECURITY DEFINER functions (migration 000002), not BYPASSRLS.
// Hand-written because sqlc cannot type set-returning function calls; still parameterised.

// OutboxEvent is one queued domain event awaiting delivery to the broker.
type OutboxEvent struct {
	ID       uuid.UUID
	TenantID uuid.UUID
	Topic    string
	Payload  json.RawMessage
	Attempts int32
}

// ClaimPendingOutbox atomically claims up to limit undelivered events (SKIP LOCKED).
func (p *Pool) ClaimPendingOutbox(ctx context.Context, limit, maxAttempts int) ([]OutboxEvent, error) {
	rows, err := p.Query(ctx, `SELECT id, tenant_id, topic, payload, attempts FROM outbox_claim_pending($1, $2)`, limit, maxAttempts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]OutboxEvent, 0, limit)
	for rows.Next() {
		var e OutboxEvent
		if err := rows.Scan(&e.ID, &e.TenantID, &e.Topic, &e.Payload, &e.Attempts); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (p *Pool) MarkOutboxPublished(ctx context.Context, id uuid.UUID) error {
	_, err := p.Exec(ctx, `SELECT outbox_mark_published($1)`, id)
	return err
}

// MarkOutboxFailed records the error and dead-letters after maxAttempts.
func (p *Pool) MarkOutboxFailed(ctx context.Context, id uuid.UUID, reason string, maxAttempts int) error {
	_, err := p.Exec(ctx, `SELECT outbox_mark_failed($1, $2, $3)`, id, reason, maxAttempts)
	return err
}

func (p *Pool) CountOutboxDeadLetters(ctx context.Context) (int64, error) {
	var n int64
	err := p.QueryRow(ctx, `SELECT outbox_dead_letter_count()`).Scan(&n)
	return n, err
}

// TenantUsageRow is a per-tenant daily rollup for the superadmin dashboard.
type TenantUsageRow struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	Day          time.Time `json:"day"`
	RequestCount int64     `json:"request_count"`
	ErrorCount   int64     `json:"error_count"`
	StudentCount int32     `json:"student_count"`
}

func (p *Pool) TenantUsageSince(ctx context.Context, since time.Time) ([]TenantUsageRow, error) {
	rows, err := p.Query(ctx, `SELECT tenant_id, day, request_count, error_count, student_count FROM tenant_usage_since($1)`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []TenantUsageRow{}
	for rows.Next() {
		var r TenantUsageRow
		if err := rows.Scan(&r.TenantID, &r.Day, &r.RequestCount, &r.ErrorCount, &r.StudentCount); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
