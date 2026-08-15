-- name: CreateAnnouncement :one
INSERT INTO announcements (tenant_id, author_id, title, body, audience_scope, audience_filter)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListAnnouncements :many
SELECT * FROM announcements
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: CreateDocument :one
INSERT INTO documents (tenant_id, student_id, type, status)
VALUES ($1, $2, $3, 'requested')
RETURNING *;

-- name: UpdateDocumentStatus :one
UPDATE documents
SET status = $2, storage_ref = COALESCE(sqlc.narg(storage_ref), storage_ref),
    generated_at = CASE WHEN $2 = 'ready' THEN now() ELSE generated_at END
WHERE tenant_id = $3 AND id = $1
RETURNING *;

-- name: GetDocument :one
SELECT * FROM documents WHERE tenant_id = $1 AND id = $2;

-- name: ListStudentDocuments :many
SELECT * FROM documents WHERE tenant_id = $1 AND student_id = $2 ORDER BY created_at DESC;

-- name: InsertAuditLog :exec
INSERT INTO audit_log (tenant_id, actor_id, action, entity, entity_id, metadata)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListAuditLogs :many
SELECT * FROM audit_log WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2;

-- name: CreateTimetableSlot :one
INSERT INTO timetable_slots (tenant_id, course_id, semester, day_of_week, start_time, end_time, room)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListTimetableForSemester :many
SELECT t.*, c.code AS course_code, c.name AS course_name
FROM timetable_slots t
JOIN courses c ON c.id = t.course_id
WHERE t.tenant_id = $1 AND t.semester = $2
ORDER BY t.day_of_week, t.start_time;

-- name: UpsertPushSubscription :one
INSERT INTO push_subscriptions (tenant_id, user_id, endpoint, p256dh, auth)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (tenant_id, endpoint)
DO UPDATE SET p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth, user_id = EXCLUDED.user_id
RETURNING *;

-- name: ListPushSubscriptionsForTenant :many
SELECT * FROM push_subscriptions WHERE tenant_id = $1;

-- name: InsertOutbox :one
INSERT INTO outbox (tenant_id, topic, payload)
VALUES ($1, $2, $3)
RETURNING *;

-- The outbox relay spans every tenant, so it cannot run inside a tenant's RLS
-- scope. It goes through the SECURITY DEFINER functions added in migration
-- 000002; sqlc cannot type set-returning functions, so those calls live in
-- internal/db/outbox.go instead of here.

-- name: CreateBulkJob :one
INSERT INTO bulk_jobs (tenant_id, job_type, status, created_by)
VALUES ($1, $2, 'pending', $3)
RETURNING *;

-- name: UpdateBulkJob :one
UPDATE bulk_jobs
SET status = $2, total_rows = $3, success_rows = $4, error_report = $5, completed_at = CASE WHEN $2 IN ('completed', 'failed') THEN now() ELSE completed_at END
WHERE tenant_id = $6 AND id = $1
RETURNING *;

-- name: GetBulkJob :one
SELECT * FROM bulk_jobs WHERE tenant_id = $1 AND id = $2;

-- name: ListBulkJobs :many
SELECT * FROM bulk_jobs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2;

-- name: UpsertTenantUsage :exec
INSERT INTO tenant_usage_daily (tenant_id, day, request_count, error_count, student_count)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (tenant_id, day)
DO UPDATE SET
  request_count = tenant_usage_daily.request_count + EXCLUDED.request_count,
  error_count = tenant_usage_daily.error_count + EXCLUDED.error_count,
  student_count = EXCLUDED.student_count;

-- name: CountStudentsForTenant :one
SELECT COUNT(*)::int AS count FROM students WHERE tenant_id = $1;
