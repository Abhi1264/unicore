package services

import (
	"context"
	"errors"
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

type CourseAttendanceSummary struct {
	CourseID    uuid.UUID `json:"course_id"`
	Total       int32     `json:"total"`
	Present     int32     `json:"present"`
	Percentage  float64   `json:"percentage"`
	Threshold   float64   `json:"threshold"`
	HasShortage bool      `json:"has_shortage"`
}

type StudentAttendanceSummary struct {
	StudentID  uuid.UUID                 `json:"student_id"`
	Threshold  float64                   `json:"threshold"`
	Courses    []CourseAttendanceSummary `json:"courses"`
	Sessions   []sqlcdb.Attendance       `json:"sessions,omitempty"`
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
		for courseID := range byCourse {
			stats, err := q.AttendanceStatsForStudentCourse(ctx, sqlcdb.AttendanceStatsForStudentCourseParams{
				TenantID:  tenantID,
				StudentID: studentID,
				CourseID:  courseID,
			})
			if err != nil {
				return err
			}
			pct := 0.0
			if stats.Total > 0 {
				pct = round2(float64(stats.Present) / float64(stats.Total) * 100)
			}
			courses = append(courses, CourseAttendanceSummary{
				CourseID:    courseID,
				Total:       stats.Total,
				Present:     stats.Present,
				Percentage:  pct,
				Threshold:   threshold,
				HasShortage: stats.Total > 0 && pct < threshold,
			})
		}

		summary = StudentAttendanceSummary{
			StudentID: studentID,
			Threshold: threshold,
			Courses:   courses,
			Sessions:  sessions,
		}
		return nil
	})
	return summary, fmtErr("attendance summary", err)
}
