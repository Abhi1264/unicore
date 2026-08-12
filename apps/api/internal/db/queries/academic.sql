-- name: CreateDepartment :one
INSERT INTO departments (tenant_id, code, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListDepartments :many
SELECT * FROM departments WHERE tenant_id = $1 ORDER BY code;

-- name: CreateCourse :one
INSERT INTO courses (tenant_id, code, name, credits, department_id, seat_cap)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetCourseByID :one
SELECT * FROM courses WHERE tenant_id = $1 AND id = $2;

-- name: ListCourses :many
SELECT * FROM courses WHERE tenant_id = $1 ORDER BY code;

-- name: LockCourseForUpdate :one
SELECT * FROM courses WHERE tenant_id = $1 AND id = $2 FOR UPDATE;

-- name: CountActiveEnrollments :one
SELECT COUNT(*)::int AS count
FROM enrollments
WHERE tenant_id = $1 AND course_id = $2 AND semester = $3 AND status = 'active';

-- name: CreateRegistrationWindow :one
INSERT INTO registration_windows (tenant_id, name, semester, opens_at, closes_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetOpenRegistrationWindow :one
SELECT * FROM registration_windows
WHERE tenant_id = $1 AND semester = $2 AND opens_at <= now() AND closes_at >= now()
ORDER BY opens_at DESC
LIMIT 1;

-- name: CreateEnrollment :one
INSERT INTO enrollments (tenant_id, student_id, course_id, semester, status, idempotency_key)
VALUES ($1, $2, $3, $4, 'active', $5)
RETURNING *;

-- name: GetEnrollmentByIdempotency :one
SELECT * FROM enrollments WHERE tenant_id = $1 AND idempotency_key = $2;

-- name: DropEnrollment :one
UPDATE enrollments SET status = 'dropped'
WHERE tenant_id = $1 AND student_id = $2 AND course_id = $3 AND semester = $4 AND status = 'active'
RETURNING *;

-- name: ListStudentEnrollments :many
SELECT * FROM enrollments
WHERE tenant_id = $1 AND student_id = $2 AND status = 'active'
ORDER BY created_at;

-- name: ListCourseRoster :many
SELECT e.*, s.roll_number, s.program, s.batch_year, u.full_name, u.email
FROM enrollments e
JOIN students s ON s.id = e.student_id
JOIN users u ON u.id = s.user_id
WHERE e.tenant_id = $1 AND e.course_id = $2 AND e.semester = $3 AND e.status = 'active'
ORDER BY s.roll_number;
