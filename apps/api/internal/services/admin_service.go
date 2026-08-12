package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/Abhi1264/unicore/api/internal/queue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AdminService struct {
	pool *db.Pool
}

func NewAdminService(pool *db.Pool) *AdminService {
	return &AdminService{pool: pool}
}

func (s *AdminService) ListTenants(ctx context.Context) ([]sqlcdb.Tenant, error) {
	tenants, err := s.pool.Platform().ListTenants(ctx)
	return tenants, fmtErr("list tenants", err)
}

func (s *AdminService) ListPendingTenants(ctx context.Context) ([]sqlcdb.Tenant, error) {
	tenants, err := s.pool.Platform().ListPendingTenants(ctx)
	return tenants, fmtErr("list pending tenants", err)
}

func (s *AdminService) ApproveTenant(ctx context.Context, tenantID uuid.UUID) (sqlcdb.Tenant, error) {
	return s.updateStatus(ctx, tenantID, "active")
}

func (s *AdminService) RejectTenant(ctx context.Context, tenantID uuid.UUID) (sqlcdb.Tenant, error) {
	return s.updateStatus(ctx, tenantID, "rejected")
}

func (s *AdminService) SuspendTenant(ctx context.Context, tenantID uuid.UUID) (sqlcdb.Tenant, error) {
	return s.updateStatus(ctx, tenantID, "suspended")
}

func (s *AdminService) ReactivateTenant(ctx context.Context, tenantID uuid.UUID) (sqlcdb.Tenant, error) {
	return s.updateStatus(ctx, tenantID, "active")
}

func (s *AdminService) updateStatus(ctx context.Context, tenantID uuid.UUID, status string) (sqlcdb.Tenant, error) {
	tenant, err := s.pool.Platform().UpdateTenantStatus(ctx, sqlcdb.UpdateTenantStatusParams{
		ID:     tenantID,
		Status: status,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlcdb.Tenant{}, ErrNotFound
	}
	return tenant, fmtErr("update tenant status", err)
}

type UpdateBrandingInput struct {
	GradingSystem          *string
	AcademicCalendarType   *string
	Branding               json.RawMessage
	GradingScale           json.RawMessage
	AttendanceThresholdPct *float64
}

func (s *AdminService) UpdateBranding(ctx context.Context, tenantID uuid.UUID, in UpdateBrandingInput) (sqlcdb.TenantConfig, error) {
	// Validation happens before the transaction opens: this payload is authored
	// by an institute admin and rendered in their users' browsers, so it is
	// untrusted from the platform's point of view.
	branding, err := validateBranding(in.Branding)
	if err != nil {
		return sqlcdb.TenantConfig{}, err
	}
	scale, err := validateGradingScale(in.GradingScale)
	if err != nil {
		return sqlcdb.TenantConfig{}, err
	}
	if in.GradingSystem != nil {
		if err := validateEnum("grading_system", *in.GradingSystem, validGradingSystems); err != nil {
			return sqlcdb.TenantConfig{}, err
		}
	}
	if in.AcademicCalendarType != nil {
		if err := validateEnum("academic_calendar_type", *in.AcademicCalendarType, validCalendarTypes); err != nil {
			return sqlcdb.TenantConfig{}, err
		}
	}
	if in.AttendanceThresholdPct != nil {
		if v := *in.AttendanceThresholdPct; v < 0 || v > 100 {
			return sqlcdb.TenantConfig{}, fmt.Errorf("%w: attendance_threshold_pct must be between 0 and 100", ErrInvalidInput)
		}
	}

	var cfg sqlcdb.TenantConfig
	err = s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var grading any
		var calendar any
		if in.GradingSystem != nil {
			grading = *in.GradingSystem
		}
		if in.AcademicCalendarType != nil {
			calendar = *in.AcademicCalendarType
		}
		var threshold pgtype.Numeric
		if in.AttendanceThresholdPct != nil {
			threshold = NumericFromFloat(*in.AttendanceThresholdPct)
		}
		var err error
		cfg, err = q.UpdateTenantConfig(ctx, sqlcdb.UpdateTenantConfigParams{
			TenantID:               tenantID,
			GradingSystem:          grading,
			AcademicCalendarType:   calendar,
			Branding:               branding,
			GradingScale:           scale,
			AttendanceThresholdPct: threshold,
		})
		return err
	})
	return cfg, fmtErr("update branding", err)
}

func (s *AdminService) GetTenantConfig(ctx context.Context, tenantID uuid.UUID) (sqlcdb.TenantConfig, error) {
	var cfg sqlcdb.TenantConfig
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		cfg, err = q.GetTenantConfig(ctx, tenantID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	return cfg, fmtErr("get tenant config", err)
}

func (s *AdminService) CreateBulkImportJob(ctx context.Context, tenantID uuid.UUID, jobType string, createdBy uuid.UUID) (sqlcdb.BulkJob, error) {
	if jobType == "" {
		return sqlcdb.BulkJob{}, ErrInvalidInput
	}
	var job sqlcdb.BulkJob
	err := s.pool.WithTenantTx(ctx, tenantID, func(ctx context.Context, _ pgx.Tx, q *sqlcdb.Queries) error {
		var err error
		job, err = q.CreateBulkJob(ctx, sqlcdb.CreateBulkJobParams{
			TenantID:  tenantID,
			JobType:   jobType,
			CreatedBy: UUID(createdBy),
		})
		if err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{
			"job_id":    job.ID,
			"tenant_id": tenantID,
			"job_type":  jobType,
		})
		_, err = q.InsertOutbox(ctx, sqlcdb.InsertOutboxParams{
			TenantID: tenantID,
			Topic:    queue.TopicBulkImport,
			Payload:  payload,
		})
		return err
	})
	return job, fmtErr("create bulk job", err)
}

func (s *AdminService) ListAuditLogs(ctx context.Context, tenantID uuid.UUID, limit int32) ([]sqlcdb.AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var out []sqlcdb.AuditLog
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.ListAuditLogs(ctx, sqlcdb.ListAuditLogsParams{
			TenantID: tenantID,
			Limit:    limit,
		})
		return err
	})
	return out, fmtErr("list audit logs", err)
}

// ListUsage aggregates across every tenant, so it goes through the platform
// helper rather than a tenant-scoped transaction. Only superadmins reach it.
func (s *AdminService) ListUsage(ctx context.Context, since time.Time) ([]db.TenantUsageRow, error) {
	if since.IsZero() {
		since = time.Now().UTC().AddDate(0, 0, -30)
	}
	rows, err := s.pool.TenantUsageSince(ctx, since.UTC())
	return rows, fmtErr("list usage", err)
}
