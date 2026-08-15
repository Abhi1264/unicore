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

-- name: ListAttendanceForCourseSession :many
SELECT a.id, a.student_id, a.status, s.roll_number, u.full_name
FROM attendance a
JOIN students s ON s.id = a.student_id
JOIN users u ON u.id = s.user_id
WHERE a.tenant_id = $1 AND a.course_id = $2 AND a.session_date = $3
ORDER BY s.roll_number;

-- name: AttendanceStatsForStudentCourse :one
SELECT
  COUNT(*)::int AS total,
  COUNT(*) FILTER (WHERE status IN ('present', 'late'))::int AS present
FROM attendance
WHERE tenant_id = $1 AND student_id = $2 AND course_id = $3;
