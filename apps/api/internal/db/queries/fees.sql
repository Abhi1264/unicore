-- name: CreateFeeHead :one
INSERT INTO fee_heads (tenant_id, name, amount, due_date, late_fee_amount, applicable_programs)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListFeeHeads :many
SELECT * FROM fee_heads WHERE tenant_id = $1 ORDER BY due_date NULLS LAST, name;

-- name: GetFeeHead :one
SELECT * FROM fee_heads WHERE tenant_id = $1 AND id = $2;

-- name: CreateFeePayment :one
INSERT INTO fee_payments (tenant_id, student_id, fee_head_id, amount, status, idempotency_key, gateway_ref)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetFeePaymentByIdempotency :one
SELECT * FROM fee_payments WHERE tenant_id = $1 AND idempotency_key = $2;

-- name: GetFeePaymentByID :one
SELECT * FROM fee_payments WHERE tenant_id = $1 AND id = $2;

-- name: MarkFeePaymentPaid :one
UPDATE fee_payments
SET status = 'paid', paid_at = now(), gateway_ref = COALESCE($3, gateway_ref)
WHERE tenant_id = $1 AND id = $2 AND status = 'pending'
RETURNING *;

-- name: ListStudentPayments :many
SELECT * FROM fee_payments WHERE tenant_id = $1 AND student_id = $2 ORDER BY created_at DESC;

-- name: ListUnpaidFeeHeadsForStudent :many
SELECT fh.*
FROM fee_heads fh
WHERE fh.tenant_id = $1
  AND (fh.applicable_programs = '{}' OR $2 = ANY(fh.applicable_programs))
  AND NOT EXISTS (
    SELECT 1 FROM fee_payments fp
    WHERE fp.tenant_id = fh.tenant_id
      AND fp.fee_head_id = fh.id
      AND fp.student_id = $3
      AND fp.status = 'paid'
  )
ORDER BY fh.due_date NULLS LAST;
