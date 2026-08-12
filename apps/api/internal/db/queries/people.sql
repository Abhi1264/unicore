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
