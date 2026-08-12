package services

import (
	"context"
	"encoding/json"

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

func (s *AnnouncementsService) Create(ctx context.Context, tenantID uuid.UUID, in CreateAnnouncementInput) (sqlcdb.Announcement, error) {
	scope := in.AudienceScope
	if scope == "" {
		scope = "all"
	}
	filter := in.AudienceFilter
	if len(filter) == 0 {
		filter = json.RawMessage(`{}`)
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

func (s *AnnouncementsService) List(ctx context.Context, tenantID uuid.UUID, limit int32) ([]sqlcdb.Announcement, error) {
	if limit <= 0 {
		limit = 50
	}
	var out []sqlcdb.Announcement
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.ListAnnouncements(ctx, sqlcdb.ListAnnouncementsParams{
			TenantID: tenantID,
			Limit:    limit,
		})
		return err
	})
	return out, fmtErr("list announcements", err)
}
