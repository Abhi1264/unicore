-- Course instructors bind a faculty member to a course for a semester.
-- Faculty may only mark attendance or enter results for courses they teach,
-- unless the course has no instructors yet (open for any faculty).

CREATE TABLE course_instructors (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  faculty_id UUID NOT NULL REFERENCES faculty(id) ON DELETE CASCADE,
  semester TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, course_id, faculty_id, semester)
);

CREATE INDEX idx_course_instructors_faculty
  ON course_instructors (tenant_id, faculty_id, semester);
CREATE INDEX idx_course_instructors_course
  ON course_instructors (tenant_id, course_id, semester);

ALTER TABLE course_instructors ENABLE ROW LEVEL SECURITY;
ALTER TABLE course_instructors FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON course_instructors
  USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
  WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON course_instructors TO unicore_app;
