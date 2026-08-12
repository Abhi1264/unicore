-- name: UpsertAttendance :one
INSERT INTO attendance (tenant_id, student_id, course_id, session_date, status, marked_by)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tenant_id, student_id, course_id, session_date)
DO UPDATE SET status = EXCLUDED.status, marked_by = EXCLUDED.marked_by
RETURNING *;

-- name: ListAttendanceForStudent :many
SELECT * FROM attendance
WHERE tenant_id = $1 AND student_id = $2
ORDER BY session_date DESC;

-- name: AttendanceStatsForStudentCourse :one
SELECT
  COUNT(*)::int AS total,
  COUNT(*) FILTER (WHERE status IN ('present', 'late'))::int AS present
FROM attendance
WHERE tenant_id = $1 AND student_id = $2 AND course_id = $3;
