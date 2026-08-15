package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AttendanceService struct {
	pool *db.Pool
}

func NewAttendanceService(pool *db.Pool) *AttendanceService {
	return &AttendanceService{pool: pool}
}

type MarkAttendanceInput struct {
	StudentID   uuid.UUID
	CourseID    uuid.UUID
	SessionDate time.Time
	Status      string // present|absent|late|excused
	MarkedBy    uuid.UUID
}

func (s *AttendanceService) MarkAttendance(ctx context.Context, tenantID uuid.UUID, in MarkAttendanceInput) (sqlcdb.Attendance, error) {
	switch in.Status {
	case "present", "absent", "late", "excused":
	default:
		return sqlcdb.Attendance{}, ErrInvalidInput
	}

	var out sqlcdb.Attendance
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.UpsertAttendance(ctx, sqlcdb.UpsertAttendanceParams{
			TenantID:    tenantID,
			StudentID:   in.StudentID,
			CourseID:    in.CourseID,
			SessionDate: DateFromTime(in.SessionDate),
			Status:      in.Status,
			MarkedBy:    UUID(in.MarkedBy),
		})
		return err
	})
	return out, fmtErr("mark attendance", err)
}

const maxAttendanceBatch = 500

type SessionMark struct {
	StudentID uuid.UUID `json:"student_id"`
	Status    string    `json:"status"`
}

type MarkSessionInput struct {
	CourseID    uuid.UUID
	SessionDate time.Time
	Marks       []SessionMark
	MarkedBy    uuid.UUID
}

func (s *AttendanceService) MarkSession(ctx context.Context, tenantID uuid.UUID, in MarkSessionInput) ([]sqlcdb.Attendance, error) {
	if len(in.Marks) == 0 || len(in.Marks) > maxAttendanceBatch {
		return nil, fmt.Errorf("%w: session must include 1-%d marks", ErrInvalidInput, maxAttendanceBatch)
	}
	out := make([]sqlcdb.Attendance, 0, len(in.Marks))
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		for _, m := range in.Marks {
			switch m.Status {
			case "present", "absent", "late", "excused":
			default:
				return ErrInvalidInput
			}
			if m.StudentID == uuid.Nil {
				return ErrInvalidInput
			}
			rec, err := q.UpsertAttendance(ctx, sqlcdb.UpsertAttendanceParams{
				TenantID:    tenantID,
				StudentID:   m.StudentID,
				CourseID:    in.CourseID,
				SessionDate: DateFromTime(in.SessionDate),
				Status:      m.Status,
				MarkedBy:    UUID(in.MarkedBy),
			})
			if err != nil {
				return err
			}
			out = append(out, rec)
		}
		return nil
	})
	return out, fmtErr("mark session", err)
}

func (s *AttendanceService) ListCourseSession(ctx context.Context, tenantID, courseID uuid.UUID, sessionDate time.Time) ([]sqlcdb.ListAttendanceForCourseSessionRow, error) {
	var out []sqlcdb.ListAttendanceForCourseSessionRow
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.ListAttendanceForCourseSession(ctx, sqlcdb.ListAttendanceForCourseSessionParams{
			TenantID:    tenantID,
			CourseID:    courseID,
			SessionDate: DateFromTime(sessionDate),
		})
		return err
	})
	return out, fmtErr("list course session", err)
}

type CourseAttendanceSummary struct {
	CourseID    uuid.UUID `json:"course_id"`
	CourseCode  string    `json:"course_code"`
	CourseName  string    `json:"course_name"`
	Total       int32     `json:"total"`
	Present     int32     `json:"present"`
	Absent      int32     `json:"absent"`
	Percentage  float64   `json:"percentage"`
	Threshold   float64   `json:"threshold"`
	HasShortage bool      `json:"has_shortage"`
}

type StudentAttendanceSummary struct {
	StudentID         uuid.UUID                 `json:"student_id"`
	Threshold         float64                   `json:"threshold"`
	OverallPercentage float64                   `json:"overall_percentage"`
	Courses           []CourseAttendanceSummary `json:"courses"`
	Sessions          []sqlcdb.Attendance       `json:"sessions,omitempty"`
}

func (s *AttendanceService) StudentAttendanceSummary(ctx context.Context, tenantID, studentID uuid.UUID) (StudentAttendanceSummary, error) {
	var summary StudentAttendanceSummary
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		threshold := 75.0
		if cfg, err := q.GetTenantConfig(ctx, tenantID); err == nil {
			if t, ok := FloatFromNumeric(cfg.AttendanceThresholdPct); ok {
				threshold = t
			}
		}

		sessions, err := q.ListAttendanceForStudent(ctx, sqlcdb.ListAttendanceForStudentParams{
			TenantID:  tenantID,
			StudentID: studentID,
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		byCourse := map[uuid.UUID]struct{}{}
		for _, a := range sessions {
			byCourse[a.CourseID] = struct{}{}
		}

		courses := make([]CourseAttendanceSummary, 0, len(byCourse))
		var totalSessions, totalPresent int32
		for courseID := range byCourse {
			stats, err := q.AttendanceStatsForStudentCourse(ctx, sqlcdb.AttendanceStatsForStudentCourseParams{
				TenantID:  tenantID,
				StudentID: studentID,
				CourseID:  courseID,
			})
			if err != nil {
				return err
			}
			course, err := q.GetCourseByID(ctx, sqlcdb.GetCourseByIDParams{TenantID: tenantID, ID: courseID})
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			pct := 0.0
			if stats.Total > 0 {
				pct = round2(float64(stats.Present) / float64(stats.Total) * 100)
			}
			absent := stats.Total - stats.Present
			if absent < 0 {
				absent = 0
			}
			totalSessions += stats.Total
			totalPresent += stats.Present
			courses = append(courses, CourseAttendanceSummary{
				CourseID:    courseID,
				CourseCode:  course.Code,
				CourseName:  course.Name,
				Total:       stats.Total,
				Present:     stats.Present,
				Absent:      absent,
				Percentage:  pct,
				Threshold:   threshold,
				HasShortage: stats.Total > 0 && pct < threshold,
			})
		}

		overall := 0.0
		if totalSessions > 0 {
			overall = round2(float64(totalPresent) / float64(totalSessions) * 100)
		}
		summary = StudentAttendanceSummary{
			StudentID:         studentID,
			Threshold:         threshold,
			OverallPercentage: overall,
			Courses:           courses,
			Sessions:          sessions,
		}
		return nil
	})
	return summary, fmtErr("attendance summary", err)
}
