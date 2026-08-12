-- name: UpsertResult :one
INSERT INTO results (tenant_id, student_id, course_id, semester, grade, grade_points, marks, submission_status, entered_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (tenant_id, student_id, course_id, semester)
DO UPDATE SET
  grade = EXCLUDED.grade,
  grade_points = EXCLUDED.grade_points,
  marks = EXCLUDED.marks,
  submission_status = EXCLUDED.submission_status,
  entered_by = EXCLUDED.entered_by,
  updated_at = now()
RETURNING *;

-- name: PublishResultsForCourseSemester :many
UPDATE results
SET submission_status = 'published', published_at = now(), updated_at = now()
WHERE tenant_id = $1 AND course_id = $2 AND semester = $3 AND submission_status IN ('draft', 'submitted')
RETURNING *;

-- name: ListPublishedResultsForStudent :many
SELECT r.*, c.code AS course_code, c.name AS course_name, c.credits
FROM results r
JOIN courses c ON c.id = r.course_id
WHERE r.tenant_id = $1 AND r.student_id = $2 AND r.submission_status = 'published'
ORDER BY r.semester, c.code;

-- name: ListPublishedResultsForStudentSemester :many
SELECT r.*, c.code AS course_code, c.name AS course_name, c.credits
FROM results r
JOIN courses c ON c.id = r.course_id
WHERE r.tenant_id = $1 AND r.student_id = $2 AND r.semester = $3 AND r.submission_status = 'published'
ORDER BY c.code;
