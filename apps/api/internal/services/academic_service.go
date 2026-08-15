package services

import (
	"context"
	"errors"
	"time"

	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AcademicService struct {
	pool *db.Pool
}

func NewAcademicService(pool *db.Pool) *AcademicService {
	return &AcademicService{pool: pool}
}

func (s *AcademicService) ListDepartments(ctx context.Context, tenantID uuid.UUID) ([]sqlcdb.Department, error) {
	var out []sqlcdb.Department
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.ListDepartments(ctx, tenantID)
		return err
	})
	return out, fmtErr("list departments", err)
}

func (s *AcademicService) CreateDepartment(ctx context.Context, tenantID uuid.UUID, code, name string) (sqlcdb.Department, error) {
	var out sqlcdb.Department
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.CreateDepartment(ctx, sqlcdb.CreateDepartmentParams{
			TenantID: tenantID,
			Code:     code,
			Name:     name,
		})
		return err
	})
	return out, fmtErr("create department", err)
}

type CreateCourseInput struct {
	Code         string
	Name         string
	Credits      float64
	DepartmentID *uuid.UUID
	SeatCap      int32
}

func (s *AcademicService) ListCourses(ctx context.Context, tenantID uuid.UUID) ([]sqlcdb.Course, error) {
	var out []sqlcdb.Course
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.ListCourses(ctx, tenantID)
		return err
	})
	return out, fmtErr("list courses", err)
}

func (s *AcademicService) CreateCourse(ctx context.Context, tenantID uuid.UUID, in CreateCourseInput) (sqlcdb.Course, error) {
	if in.SeatCap <= 0 {
		in.SeatCap = 60
	}
	if in.Credits <= 0 {
		in.Credits = 3
	}
	var out sqlcdb.Course
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.CreateCourse(ctx, sqlcdb.CreateCourseParams{
			TenantID:     tenantID,
			Code:         in.Code,
			Name:         in.Name,
			Credits:      NumericFromFloat(in.Credits),
			DepartmentID: UUIDPtr(in.DepartmentID),
			SeatCap:      in.SeatCap,
		})
		return err
	})
	return out, fmtErr("create course", err)
}

type EnrollStudentInput struct {
	StudentID      uuid.UUID
	CourseID       uuid.UUID
	Semester       string
	IdempotencyKey string
}

func (s *AcademicService) EnrollStudent(ctx context.Context, tenantID uuid.UUID, in EnrollStudentInput) (sqlcdb.Enrollment, error) {
	if in.Semester == "" {
		return sqlcdb.Enrollment{}, ErrInvalidInput
	}

	var enrollment sqlcdb.Enrollment
	err := s.pool.WithTenantTx(ctx, tenantID, func(ctx context.Context, _ pgx.Tx, q *sqlcdb.Queries) error {
		if _, err := q.GetOpenRegistrationWindow(ctx, sqlcdb.GetOpenRegistrationWindowParams{
			TenantID: tenantID,
			Semester: in.Semester,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrRegistrationClosed
			}
			return err
		}

		if in.IdempotencyKey != "" {
			existing, err := q.GetEnrollmentByIdempotency(ctx, sqlcdb.GetEnrollmentByIdempotencyParams{
				TenantID:       tenantID,
				IdempotencyKey: Text(in.IdempotencyKey),
			})
			if err == nil {
				enrollment = existing
				return nil
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}

		// The row lock is what makes the seat cap safe: every concurrent
		// registration for this course serialises here, so the count below cannot
		// go stale between reading it and inserting the enrollment.
		course, err := q.LockCourseForUpdate(ctx, sqlcdb.LockCourseForUpdateParams{
			TenantID: tenantID,
			ID:       in.CourseID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}

		count, err := q.CountActiveEnrollments(ctx, sqlcdb.CountActiveEnrollmentsParams{
			TenantID: tenantID,
			CourseID: in.CourseID,
			Semester: in.Semester,
		})
		if err != nil {
			return err
		}
		if count >= course.SeatCap {
			return ErrSeatFull
		}

		var idem pgtype.Text
		if in.IdempotencyKey != "" {
			idem = Text(in.IdempotencyKey)
		}
		enrollment, err = q.CreateEnrollment(ctx, sqlcdb.CreateEnrollmentParams{
			TenantID:       tenantID,
			StudentID:      in.StudentID,
			CourseID:       in.CourseID,
			Semester:       in.Semester,
			IdempotencyKey: idem,
		})
		if isUniqueViolation(err) {
			// Already enrolled in this course/semester, or the same idempotency key
			// was replayed between the lookup above and this insert.
			return ErrConflict
		}
		if isForeignKeyViolation(err) {
			return ErrNotFound
		}
		return err
	})
	return enrollment, fmtErr("enroll student", err)
}

func (s *AcademicService) DropEnrollment(ctx context.Context, tenantID, studentID, courseID uuid.UUID, semester string) (sqlcdb.Enrollment, error) {
	var out sqlcdb.Enrollment
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.DropEnrollment(ctx, sqlcdb.DropEnrollmentParams{
			TenantID:  tenantID,
			StudentID: studentID,
			CourseID:  courseID,
			Semester:  semester,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	return out, fmtErr("drop enrollment", err)
}

func (s *AcademicService) ListRoster(ctx context.Context, tenantID, courseID uuid.UUID, semester string) ([]sqlcdb.ListCourseRosterRow, error) {
	var out []sqlcdb.ListCourseRosterRow
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.ListCourseRoster(ctx, sqlcdb.ListCourseRosterParams{
			TenantID: tenantID,
			CourseID: courseID,
			Semester: semester,
		})
		return err
	})
	return out, fmtErr("list roster", err)
}

type MyEnrollment struct {
	ID         uuid.UUID `json:"id"`
	CourseID   uuid.UUID `json:"course_id"`
	CourseCode string    `json:"course_code"`
	CourseName string    `json:"course_name"`
	Credits    float64   `json:"credits"`
	SeatCap    int32     `json:"seat_cap"`
	Semester   string    `json:"semester"`
	Status     string    `json:"status"`
}

func (s *AcademicService) ListMyEnrollments(ctx context.Context, tenantID, studentID uuid.UUID) ([]MyEnrollment, error) {
	var out []MyEnrollment
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		rows, err := q.ListStudentEnrollmentsWithCourse(ctx, sqlcdb.ListStudentEnrollmentsWithCourseParams{
			TenantID:  tenantID,
			StudentID: studentID,
		})
		if err != nil {
			return err
		}
		out = make([]MyEnrollment, 0, len(rows))
		for _, r := range rows {
			credits, _ := FloatFromNumeric(r.Credits)
			out = append(out, MyEnrollment{
				ID:         r.ID,
				CourseID:   r.CourseID,
				CourseCode: r.CourseCode,
				CourseName: r.CourseName,
				Credits:    credits,
				SeatCap:    r.SeatCap,
				Semester:   r.Semester,
				Status:     r.Status,
			})
		}
		return nil
	})
	return out, fmtErr("list enrollments", err)
}

func (s *AcademicService) ListFaculty(ctx context.Context, tenantID uuid.UUID) ([]sqlcdb.ListFacultyRow, error) {
	var out []sqlcdb.ListFacultyRow
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.ListFaculty(ctx, tenantID)
		return err
	})
	return out, fmtErr("list faculty", err)
}

func (s *AcademicService) AssignInstructor(ctx context.Context, tenantID, courseID, facultyID uuid.UUID, semester string) (sqlcdb.CourseInstructor, error) {
	var out sqlcdb.CourseInstructor
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		if _, err := q.GetCourseByID(ctx, sqlcdb.GetCourseByIDParams{TenantID: tenantID, ID: courseID}); err != nil {
			return err
		}
		var err error
		out, err = q.AssignCourseInstructor(ctx, sqlcdb.AssignCourseInstructorParams{
			TenantID:  tenantID,
			CourseID:  courseID,
			FacultyID: facultyID,
			Semester:  semester,
		})
		return err
	})
	return out, fmtErr("assign instructor", err)
}

func (s *AcademicService) RemoveInstructor(ctx context.Context, tenantID, courseID, facultyID uuid.UUID, semester string) error {
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		return q.RemoveCourseInstructor(ctx, sqlcdb.RemoveCourseInstructorParams{
			TenantID:  tenantID,
			CourseID:  courseID,
			FacultyID: facultyID,
			Semester:  semester,
		})
	})
	return fmtErr("remove instructor", err)
}

func (s *AcademicService) ListInstructors(ctx context.Context, tenantID, courseID uuid.UUID, semester string) ([]sqlcdb.ListCourseInstructorsRow, error) {
	var out []sqlcdb.ListCourseInstructorsRow
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.ListCourseInstructors(ctx, sqlcdb.ListCourseInstructorsParams{
			TenantID: tenantID,
			CourseID: courseID,
			Semester: semester,
		})
		return err
	})
	return out, fmtErr("list instructors", err)
}

func (s *AcademicService) ListCoursesForUser(ctx context.Context, tenantID, userID uuid.UUID, role string) ([]sqlcdb.Course, error) {
	if role != "faculty" {
		return s.ListCourses(ctx, tenantID)
	}
	var assigned []sqlcdb.Course
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		assigned, err = q.ListCoursesForFacultyUser(ctx, sqlcdb.ListCoursesForFacultyUserParams{
			TenantID: tenantID,
			UserID:   userID,
		})
		return err
	})
	if err != nil {
		return nil, fmtErr("list faculty courses", err)
	}
	if len(assigned) > 0 {
		return assigned, nil
	}
	return s.ListCourses(ctx, tenantID)
}

// AssertFacultyTeaches is a no-op when the course has no instructors assigned.
func (s *AcademicService) AssertFacultyTeaches(ctx context.Context, tenantID, userID, courseID uuid.UUID, semester string) error {
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		fac, err := q.GetFacultyByUserID(ctx, sqlcdb.GetFacultyByUserIDParams{TenantID: tenantID, UserID: userID})
		if err != nil {
			return ErrForbidden
		}
		if semester != "" {
			n, err := q.CountCourseInstructors(ctx, sqlcdb.CountCourseInstructorsParams{
				TenantID: tenantID, CourseID: courseID, Semester: semester,
			})
			if err != nil {
				return err
			}
			if n == 0 {
				return nil
			}
			ok, err := q.FacultyTeachesCourseSemester(ctx, sqlcdb.FacultyTeachesCourseSemesterParams{
				TenantID: tenantID, CourseID: courseID, Semester: semester, FacultyID: fac.ID,
			})
			if err != nil {
				return err
			}
			if !ok {
				return ErrForbidden
			}
			return nil
		}
		n, err := q.CountCourseInstructorsAnySemester(ctx, sqlcdb.CountCourseInstructorsAnySemesterParams{
			TenantID: tenantID, CourseID: courseID,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		ok, err := q.FacultyTeachesCourse(ctx, sqlcdb.FacultyTeachesCourseParams{
			TenantID: tenantID, CourseID: courseID, FacultyID: fac.ID,
		})
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		return nil
	})
	return fmtErr("assert faculty teaches", err)
}

type CreateTimetableSlotInput struct {
	CourseID  uuid.UUID
	Semester  string
	DayOfWeek int32
	StartHour int
	StartMin  int
	EndHour   int
	EndMin    int
	Room      string
}

func (s *AcademicService) ListTimetable(ctx context.Context, tenantID uuid.UUID, semester string) ([]sqlcdb.ListTimetableForSemesterRow, error) {
	var out []sqlcdb.ListTimetableForSemesterRow
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.ListTimetableForSemester(ctx, sqlcdb.ListTimetableForSemesterParams{
			TenantID: tenantID,
			Semester: semester,
		})
		return err
	})
	return out, fmtErr("list timetable", err)
}

func (s *AcademicService) CreateTimetableSlot(ctx context.Context, tenantID uuid.UUID, in CreateTimetableSlotInput) (sqlcdb.TimetableSlot, error) {
	var out sqlcdb.TimetableSlot
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.CreateTimetableSlot(ctx, sqlcdb.CreateTimetableSlotParams{
			TenantID:  tenantID,
			CourseID:  in.CourseID,
			Semester:  in.Semester,
			DayOfWeek: in.DayOfWeek,
			StartTime: TimeOfDay(in.StartHour, in.StartMin, 0),
			EndTime:   TimeOfDay(in.EndHour, in.EndMin, 0),
			Room:      TextOrEmpty(in.Room),
		})
		return err
	})
	return out, fmtErr("create timetable slot", err)
}

type CreateRegistrationWindowInput struct {
	Name     string
	Semester string
	OpensAt  time.Time
	ClosesAt time.Time
}

func (s *AcademicService) CreateRegistrationWindow(ctx context.Context, tenantID uuid.UUID, in CreateRegistrationWindowInput) (sqlcdb.RegistrationWindow, error) {
	var out sqlcdb.RegistrationWindow
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.CreateRegistrationWindow(ctx, sqlcdb.CreateRegistrationWindowParams{
			TenantID: tenantID,
			Name:     in.Name,
			Semester: in.Semester,
			OpensAt:  in.OpensAt,
			ClosesAt: in.ClosesAt,
		})
		return err
	})
	return out, fmtErr("create registration window", err)
}

func (s *AcademicService) GetOpenRegistrationWindow(ctx context.Context, tenantID uuid.UUID, semester string) (sqlcdb.RegistrationWindow, error) {
	var out sqlcdb.RegistrationWindow
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.GetOpenRegistrationWindow(ctx, sqlcdb.GetOpenRegistrationWindowParams{
			TenantID: tenantID,
			Semester: semester,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	return out, fmtErr("get open registration window", err)
}
