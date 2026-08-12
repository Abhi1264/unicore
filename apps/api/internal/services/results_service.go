package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Abhi1264/unicore/api/internal/cache"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/Abhi1264/unicore/api/internal/metrics"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

const resultsCacheTTL = 5 * time.Minute

type ResultsService struct {
	pool  *db.Pool
	cache *cache.Client
	sf    singleflight.Group
}

func NewResultsService(pool *db.Pool, cacheClient *cache.Client) *ResultsService {
	return &ResultsService{pool: pool, cache: cacheClient}
}

type ResultCourseRow struct {
	ID               uuid.UUID  `json:"id"`
	CourseID         uuid.UUID  `json:"course_id"`
	CourseCode       string     `json:"course_code"`
	CourseName       string     `json:"course_name"`
	Credits          float64    `json:"credits"`
	Semester         string     `json:"semester"`
	Grade            string     `json:"grade"`
	GradePoints      *float64   `json:"grade_points,omitempty"`
	Marks            *float64   `json:"marks,omitempty"`
	SubmissionStatus string     `json:"submission_status"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
}

type StudentResultsResponse struct {
	StudentID        uuid.UUID         `json:"student_id"`
	Semester         string            `json:"semester,omitempty"`
	GradingSystem    string            `json:"grading_system"`
	Rows             []ResultCourseRow `json:"rows"`
	CumulativeValue  float64           `json:"cumulative_value"`
	CumulativeDisplay string           `json:"cumulative_display"`
}

type EnterResultInput struct {
	StudentID   uuid.UUID
	CourseID    uuid.UUID
	Semester    string
	Grade       string
	GradePoints *float64
	Marks       *float64
	Status      string // draft|submitted
	EnteredBy   uuid.UUID
}

func (s *ResultsService) GetStudentResults(ctx context.Context, tenantID, studentID uuid.UUID, semester *string) (StudentResultsResponse, error) {
	semKey := "all"
	if semester != nil && *semester != "" {
		semKey = *semester
	}
	cacheKey := fmt.Sprintf("tenant:%s:student:%s:results:%s", tenantID, studentID, semKey)

	if s.cache != nil && s.cache.Available() {
		if raw, err := s.cache.Get(ctx, cacheKey); err == nil {
			metrics.ResultsCacheHits.Inc()
			var out StudentResultsResponse
			if json.Unmarshal([]byte(raw), &out) == nil {
				return out, nil
			}
		} else if errors.Is(err, cache.Miss) {
			metrics.ResultsCacheMisses.Inc()
		} else if !errors.Is(err, cache.ErrUnavailable) {
			metrics.ResultsCacheMisses.Inc()
		}
	}

	v, err, _ := s.sf.Do(cacheKey, func() (any, error) {
		out, err := s.loadStudentResults(ctx, tenantID, studentID, semester)
		if err != nil {
			return nil, err
		}
		if s.cache != nil {
			if b, err := json.Marshal(out); err == nil {
				_ = s.cache.Set(ctx, cacheKey, string(b), resultsCacheTTL)
			}
		}
		return out, nil
	})
	if err != nil {
		return StudentResultsResponse{}, err
	}
	return v.(StudentResultsResponse), nil
}

func (s *ResultsService) loadStudentResults(ctx context.Context, tenantID, studentID uuid.UUID, semester *string) (StudentResultsResponse, error) {
	var (
		rows            []ResultCourseRow
		gradingSystem   = "cgpa"
		inputs          []GradeInput
	)

	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		if cfg, err := q.GetTenantConfig(ctx, tenantID); err == nil {
			gradingSystem = cfg.GradingSystem
		}

		if semester != nil && *semester != "" {
			list, err := q.ListPublishedResultsForStudentSemester(ctx, sqlcdb.ListPublishedResultsForStudentSemesterParams{
				TenantID:  tenantID,
				StudentID: studentID,
				Semester:  *semester,
			})
			if err != nil {
				return err
			}
			rows = make([]ResultCourseRow, 0, len(list))
			for _, r := range list {
				row, gi := mapSemesterResult(r)
				rows = append(rows, row)
				inputs = append(inputs, gi)
			}
			return nil
		}

		list, err := q.ListPublishedResultsForStudent(ctx, sqlcdb.ListPublishedResultsForStudentParams{
			TenantID:  tenantID,
			StudentID: studentID,
		})
		if err != nil {
			return err
		}
		rows = make([]ResultCourseRow, 0, len(list))
		for _, r := range list {
			row, gi := mapAllResult(r)
			rows = append(rows, row)
			inputs = append(inputs, gi)
		}
		return nil
	})
	if err != nil {
		return StudentResultsResponse{}, fmtErr("load results", err)
	}

	value, display := ComputeCumulative(gradingSystem, inputs)
	out := StudentResultsResponse{
		StudentID:         studentID,
		GradingSystem:     gradingSystem,
		Rows:              rows,
		CumulativeValue:   value,
		CumulativeDisplay: display,
	}
	if semester != nil {
		out.Semester = *semester
	}
	return out, nil
}

func mapAllResult(r sqlcdb.ListPublishedResultsForStudentRow) (ResultCourseRow, GradeInput) {
	credits, _ := FloatFromNumeric(r.Credits)
	gp, gpOK := FloatFromNumeric(r.GradePoints)
	marks, marksOK := FloatFromNumeric(r.Marks)
	row := ResultCourseRow{
		ID:               r.ID,
		CourseID:         r.CourseID,
		CourseCode:       r.CourseCode,
		CourseName:       r.CourseName,
		Credits:          credits,
		Semester:         r.Semester,
		Grade:            r.Grade,
		SubmissionStatus: r.SubmissionStatus,
	}
	gi := GradeInput{Grade: r.Grade, Credits: credits}
	if gpOK {
		row.GradePoints = &gp
		gi.GradePoints = &gp
	}
	if marksOK {
		row.Marks = &marks
		gi.Marks = &marks
	}
	if r.PublishedAt.Valid {
		t := r.PublishedAt.Time
		row.PublishedAt = &t
	}
	return row, gi
}

func mapSemesterResult(r sqlcdb.ListPublishedResultsForStudentSemesterRow) (ResultCourseRow, GradeInput) {
	credits, _ := FloatFromNumeric(r.Credits)
	gp, gpOK := FloatFromNumeric(r.GradePoints)
	marks, marksOK := FloatFromNumeric(r.Marks)
	row := ResultCourseRow{
		ID:               r.ID,
		CourseID:         r.CourseID,
		CourseCode:       r.CourseCode,
		CourseName:       r.CourseName,
		Credits:          credits,
		Semester:         r.Semester,
		Grade:            r.Grade,
		SubmissionStatus: r.SubmissionStatus,
	}
	gi := GradeInput{Grade: r.Grade, Credits: credits}
	if gpOK {
		row.GradePoints = &gp
		gi.GradePoints = &gp
	}
	if marksOK {
		row.Marks = &marks
		gi.Marks = &marks
	}
	if r.PublishedAt.Valid {
		t := r.PublishedAt.Time
		row.PublishedAt = &t
	}
	return row, gi
}

func (s *ResultsService) EnterResult(ctx context.Context, tenantID uuid.UUID, in EnterResultInput) (sqlcdb.Result, error) {
	status := in.Status
	if status == "" {
		status = "draft"
	}
	if status != "draft" && status != "submitted" {
		return sqlcdb.Result{}, ErrInvalidInput
	}

	var result sqlcdb.Result
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		result, err = q.UpsertResult(ctx, sqlcdb.UpsertResultParams{
			TenantID:         tenantID,
			StudentID:        in.StudentID,
			CourseID:         in.CourseID,
			Semester:         in.Semester,
			Grade:            in.Grade,
			GradePoints:      NumericFromFloatPtr(in.GradePoints),
			Marks:            NumericFromFloatPtr(in.Marks),
			SubmissionStatus: status,
			EnteredBy:        UUID(in.EnteredBy),
		})
		if err != nil {
			return err
		}
		meta, _ := json.Marshal(map[string]any{
			"student_id": in.StudentID,
			"course_id":  in.CourseID,
			"semester":   in.Semester,
			"grade":      in.Grade,
			"status":     status,
		})
		return q.InsertAuditLog(ctx, sqlcdb.InsertAuditLogParams{
			TenantID: tenantID,
			ActorID:  UUID(in.EnteredBy),
			Action:   "results.enter",
			Entity:   "result",
			EntityID: UUID(result.ID),
			Metadata: meta,
		})
	})
	if err != nil {
		return sqlcdb.Result{}, fmtErr("enter result", err)
	}
	return result, nil
}

func (s *ResultsService) PublishCourseResults(ctx context.Context, tenantID, courseID uuid.UUID, semester string, actorID uuid.UUID) ([]sqlcdb.Result, error) {
	var published []sqlcdb.Result
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		published, err = q.PublishResultsForCourseSemester(ctx, sqlcdb.PublishResultsForCourseSemesterParams{
			TenantID: tenantID,
			CourseID: courseID,
			Semester: semester,
		})
		if err != nil {
			return err
		}
		meta, _ := json.Marshal(map[string]any{
			"course_id": courseID,
			"semester":  semester,
			"count":     len(published),
		})
		if err := q.InsertAuditLog(ctx, sqlcdb.InsertAuditLogParams{
			TenantID: tenantID,
			ActorID:  UUID(actorID),
			Action:   "results.publish",
			Entity:   "course",
			EntityID: UUID(courseID),
			Metadata: meta,
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, fmtErr("publish results", err)
	}

	s.invalidateResultsCache(ctx, tenantID, published)
	return published, nil
}

func (s *ResultsService) invalidateResultsCache(ctx context.Context, tenantID uuid.UUID, published []sqlcdb.Result) {
	if s.cache == nil {
		return
	}
	seen := map[string]struct{}{}
	keys := make([]string, 0, len(published)*2)
	for _, r := range published {
		allKey := fmt.Sprintf("tenant:%s:student:%s:results:all", tenantID, r.StudentID)
		semKey := fmt.Sprintf("tenant:%s:student:%s:results:%s", tenantID, r.StudentID, r.Semester)
		for _, k := range []string{allKey, semKey} {
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			keys = append(keys, k)
		}
	}
	if len(keys) > 0 {
		_ = s.cache.Del(ctx, keys...)
	}
}
