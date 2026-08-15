-- name: CreateStudent :one
INSERT INTO students (tenant_id, user_id, roll_number, program, batch_year, department_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetStudentByUserID :one
SELECT * FROM students WHERE tenant_id = $1 AND user_id = $2;

-- name: GetStudentByID :one
SELECT * FROM students WHERE tenant_id = $1 AND id = $2;

-- name: ListStudents :many
SELECT * FROM students WHERE tenant_id = $1 ORDER BY roll_number;

-- name: CreateFaculty :one
INSERT INTO faculty (tenant_id, user_id, department_id, employee_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetFacultyByUserID :one
SELECT * FROM faculty WHERE tenant_id = $1 AND user_id = $2;

-- name: ListFaculty :many
SELECT f.id, f.user_id, f.department_id, f.employee_id, u.full_name, u.email
FROM faculty f
JOIN users u ON u.id = f.user_id
WHERE f.tenant_id = $1
ORDER BY u.full_name;

-- name: AssignCourseInstructor :one
INSERT INTO course_instructors (tenant_id, course_id, faculty_id, semester)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id, course_id, faculty_id, semester) DO UPDATE SET semester = EXCLUDED.semester
RETURNING *;

-- name: RemoveCourseInstructor :exec
DELETE FROM course_instructors
WHERE tenant_id = $1 AND course_id = $2 AND faculty_id = $3 AND semester = $4;

-- name: ListCourseInstructors :many
SELECT ci.id, ci.course_id, ci.faculty_id, ci.semester, ci.created_at,
       u.full_name, u.email, f.employee_id
FROM course_instructors ci
JOIN faculty f ON f.id = ci.faculty_id
JOIN users u ON u.id = f.user_id
WHERE ci.tenant_id = $1 AND ci.course_id = $2 AND ci.semester = $3
ORDER BY u.full_name;

-- name: CountCourseInstructors :one
SELECT COUNT(*)::int AS count
FROM course_instructors
WHERE tenant_id = $1 AND course_id = $2 AND semester = $3;

-- name: CountCourseInstructorsAnySemester :one
SELECT COUNT(*)::int AS count
FROM course_instructors
WHERE tenant_id = $1 AND course_id = $2;

-- name: FacultyTeachesCourseSemester :one
SELECT EXISTS (
  SELECT 1 FROM course_instructors ci
  WHERE ci.tenant_id = $1 AND ci.course_id = $2 AND ci.semester = $3 AND ci.faculty_id = $4
)::bool AS teaches;

-- name: FacultyTeachesCourse :one
SELECT EXISTS (
  SELECT 1 FROM course_instructors ci
  WHERE ci.tenant_id = $1 AND ci.course_id = $2 AND ci.faculty_id = $3
)::bool AS teaches;

-- name: ListCoursesForFacultyUser :many
SELECT DISTINCT c.*
FROM courses c
JOIN course_instructors ci ON ci.course_id = c.id AND ci.tenant_id = c.tenant_id
JOIN faculty f ON f.id = ci.faculty_id
WHERE c.tenant_id = $1 AND f.user_id = $2
ORDER BY c.code;
