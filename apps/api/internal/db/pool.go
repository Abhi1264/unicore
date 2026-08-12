package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrRLSBypass is returned when the connection role can read past row-level security.
var ErrRLSBypass = errors.New(
	"database role can bypass row-level security, which disables all tenant isolation; " +
		"connect as a NOSUPERUSER NOBYPASSRLS role (see infra/postgres/init.sh) and run " +
		"migrations separately via DATABASE_MIGRATE_URL",
)

type Pool struct {
	*pgxpool.Pool
}

func NewPool(ctx context.Context, databaseURL string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 50
	cfg.MinConns = 5
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Pool{Pool: pool}, nil
}

// AssertRLSEnforced verifies the runtime role cannot bypass RLS.
func (p *Pool) AssertRLSEnforced(ctx context.Context) error {
	var isSuper, bypassRLS bool
	err := p.QueryRow(ctx, `
		SELECT rolsuper, rolbypassrls
		FROM pg_roles
		WHERE rolname = current_user
	`).Scan(&isSuper, &bypassRLS)
	if err != nil {
		return fmt.Errorf("inspect database role: %w", err)
	}
	if isSuper || bypassRLS {
		return ErrRLSBypass
	}
	return nil
}

// WithTenant runs fn in a transaction after SET LOCAL app.tenant_id (same tx required for RLS).
func (p *Pool) WithTenant(ctx context.Context, tenantID uuid.UUID, fn func(ctx context.Context, q *sqlcdb.Queries) error) error {
	return p.WithTenantTx(ctx, tenantID, func(ctx context.Context, _ pgx.Tx, q *sqlcdb.Queries) error {
		return fn(ctx, q)
	})
}

func (p *Pool) WithTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(ctx context.Context, tx pgx.Tx, q *sqlcdb.Queries) error) error {
	// Reject nil UUID so a missing tenant is an error, not an empty scoped result.
	if tenantID == uuid.Nil {
		return fmt.Errorf("tenant scope required")
	}

	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("set tenant: %w", err)
	}

	q := sqlcdb.New(tx)
	if err := fn(ctx, tx, q); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Platform returns queries outside tenant RLS (tenants table, SECURITY DEFINER helpers).
func (p *Pool) Platform() *sqlcdb.Queries {
	return sqlcdb.New(p.Pool)
}
