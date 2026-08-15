package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/google/uuid"
)

type AnnouncementsService struct {
	pool *db.Pool
}

func NewAnnouncementsService(pool *db.Pool) *AnnouncementsService {
	return &AnnouncementsService{pool: pool}
}

type CreateAnnouncementInput struct {
	AuthorID       uuid.UUID
	Title          string
	Body           string
	AudienceScope  string // all|program|batch|course
	AudienceFilter json.RawMessage
}

type audienceFilter struct {
	Program   string `json:"program"`
	BatchYear int    `json:"batch_year"`
	CourseID  string `json:"course_id"`
}

func (s *AnnouncementsService) Create(ctx context.Context, tenantID uuid.UUID, in CreateAnnouncementInput) (sqlcdb.Announcement, error) {
	scope := in.AudienceScope
	if scope == "" {
		scope = "all"
	}
	filter := in.AudienceFilter
	if len(filter) == 0 {
		filter = json.RawMessage(`{}`)
	}
	if err := validateAudience(scope, filter); err != nil {
		return sqlcdb.Announcement{}, err
	}

	var ann sqlcdb.Announcement
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		ann, err = q.CreateAnnouncement(ctx, sqlcdb.CreateAnnouncementParams{
			TenantID:       tenantID,
			AuthorID:       in.AuthorID,
			Title:          in.Title,
			Body:           in.Body,
			AudienceScope:  scope,
			AudienceFilter: filter,
		})
		return err
	})
	return ann, fmtErr("create announcement", err)
}

func validateAudience(scope string, raw json.RawMessage) error {
	var f audienceFilter
	if err := json.Unmarshal(raw, &f); err != nil {
		return ErrInvalidInput
	}
	switch scope {
	case "all":
		return nil
	case "program":
		if f.Program == "" {
			return fmt.Errorf("%w: program is required", ErrInvalidInput)
		}
		return nil
	case "batch":
		if f.BatchYear < 1990 || f.BatchYear > 2100 {
			return fmt.Errorf("%w: batch_year must be a valid year", ErrInvalidInput)
		}
		return nil
	case "course":
		if _, err := uuid.Parse(f.CourseID); err != nil {
			return fmt.Errorf("%w: course_id must be a UUID", ErrInvalidInput)
		}
		return nil
	default:
		return fmt.Errorf("%w: audience_scope must be all, program, batch, or course", ErrInvalidInput)
	}
}

func (s *AnnouncementsService) ListForViewer(ctx context.Context, tenantID, userID uuid.UUID, role auth.Role, limit int32) ([]sqlcdb.Announcement, error) {
	if limit <= 0 {
		limit = 50
	}
	var out []sqlcdb.Announcement
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		list, err := q.ListAnnouncements(ctx, sqlcdb.ListAnnouncementsParams{
			TenantID: tenantID,
			Limit:    limit,
		})
		if err != nil {
			return err
		}
		if role != auth.RoleStudent {
			out = list
			return nil
		}
		st, err := q.GetStudentByUserID(ctx, sqlcdb.GetStudentByUserIDParams{TenantID: tenantID, UserID: userID})
		if err != nil {
			return err
		}
		enrs, err := q.ListStudentEnrollments(ctx, sqlcdb.ListStudentEnrollmentsParams{
			TenantID: tenantID, StudentID: st.ID,
		})
		if err != nil {
			return err
		}
		courses := map[string]struct{}{}
		for _, e := range enrs {
			courses[e.CourseID.String()] = struct{}{}
		}
		out = make([]sqlcdb.Announcement, 0, len(list))
		for _, a := range list {
			if announcementVisible(a, st, courses) {
				out = append(out, a)
			}
		}
		return nil
	})
	return out, fmtErr("list announcements", err)
}

func announcementVisible(a sqlcdb.Announcement, st sqlcdb.Student, courses map[string]struct{}) bool {
	switch a.AudienceScope {
	case "all", "":
		return true
	case "program":
		var f audienceFilter
		_ = json.Unmarshal(a.AudienceFilter, &f)
		return f.Program != "" && f.Program == st.Program
	case "batch":
		var f audienceFilter
		_ = json.Unmarshal(a.AudienceFilter, &f)
		return f.BatchYear != 0 && int32(f.BatchYear) == st.BatchYear
	case "course":
		var f audienceFilter
		_ = json.Unmarshal(a.AudienceFilter, &f)
		_, ok := courses[f.CourseID]
		return ok
	default:
		return false
	}
}
