package db_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Integration tests require DATABASE_URL for the *runtime* role (not superuser):
//
//	DATABASE_URL=postgres://unicore_app:...@localhost:5432/unicore go test ./internal/db/...

func testPool(t *testing.T) *db.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ensureMigrated(t)
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func makeTenant(t *testing.T, pool *db.Pool, prefix string) sqlcdb.Tenant {
	t.Helper()
	ctx := context.Background()
	tenant, err := pool.Platform().CreateTenant(ctx, sqlcdb.CreateTenantParams{
		Slug: prefix + uuid.NewString()[:8],
		Name: "Test " + prefix,
	})
	if err != nil {
		t.Fatalf("create tenant %s: %v", prefix, err)
	}
	return tenant
}

// TestRuntimeRoleCannotBypassRLS guards every other isolation test.
func TestRuntimeRoleCannotBypassRLS(t *testing.T) {
	pool := testPool(t)
	if err := pool.AssertRLSEnforced(context.Background()); err != nil {
		t.Fatalf("runtime role can bypass row-level security: %v", err)
	}
}

// TestRLSBlocksUnscopedRead uses a query with no tenant_id predicate so only RLS can filter.
func TestRLSBlocksUnscopedRead(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	tenantA := makeTenant(t, pool, "rls-a-")
	tenantB := makeTenant(t, pool, "rls-b-")

	if err := pool.WithTenant(ctx, tenantA.ID, func(ctx context.Context, q *sqlcdb.Queries) error {
		_, err := q.CreateDepartment(ctx, sqlcdb.CreateDepartmentParams{
			TenantID: tenantA.ID, Code: "CSE", Name: "Computer Science",
		})
		return err
	}); err != nil {
		t.Fatalf("seed tenant A: %v", err)
	}

	var visible int
	if err := pool.WithTenantTx(ctx, tenantB.ID, func(ctx context.Context, tx pgx.Tx, _ *sqlcdb.Queries) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM departments`).Scan(&visible)
	}); err != nil {
		t.Fatal(err)
	}
	if visible != 0 {
		t.Fatalf("tenant B saw %d departments through an unscoped query; RLS is not filtering", visible)
	}

	if err := pool.WithTenantTx(ctx, tenantA.ID, func(ctx context.Context, tx pgx.Tx, _ *sqlcdb.Queries) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM departments`).Scan(&visible)
	}); err != nil {
		t.Fatal(err)
	}
	if visible != 1 {
		t.Fatalf("tenant A expected to see its own 1 department, saw %d", visible)
	}
}

// TestRLSBlocksCrossTenantFetchByID covers the "stolen id" case.
func TestRLSBlocksCrossTenantFetchByID(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	tenantA := makeTenant(t, pool, "rls-idr-a-")
	tenantB := makeTenant(t, pool, "rls-idr-b-")

	var deptID uuid.UUID
	if err := pool.WithTenant(ctx, tenantA.ID, func(ctx context.Context, q *sqlcdb.Queries) error {
		d, err := q.CreateDepartment(ctx, sqlcdb.CreateDepartmentParams{
			TenantID: tenantA.ID, Code: "ECE", Name: "Electronics",
		})
		deptID = d.ID
		return err
	}); err != nil {
		t.Fatalf("seed tenant A: %v", err)
	}

	var found int
	if err := pool.WithTenantTx(ctx, tenantB.ID, func(ctx context.Context, tx pgx.Tx, _ *sqlcdb.Queries) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM departments WHERE id = $1`, deptID).Scan(&found)
	}); err != nil {
		t.Fatal(err)
	}
	if found != 0 {
		t.Fatal("tenant B fetched tenant A's department by primary key")
	}
}

// TestRLSBlocksCrossTenantWrite ensures WITH CHECK blocks foreign tenant_id stamps.
func TestRLSBlocksCrossTenantWrite(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	tenantA := makeTenant(t, pool, "rls-w-a-")
	tenantB := makeTenant(t, pool, "rls-w-b-")

	err := pool.WithTenant(ctx, tenantB.ID, func(ctx context.Context, q *sqlcdb.Queries) error {
		_, err := q.CreateDepartment(ctx, sqlcdb.CreateDepartmentParams{
			TenantID: tenantA.ID, Code: "SMUGGLED", Name: "Should not exist",
		})
		return err
	})
	if err == nil {
		t.Fatal("tenant B inserted a row belonging to tenant A")
	}

	var leaked int
	if err := pool.WithTenantTx(ctx, tenantA.ID, func(ctx context.Context, tx pgx.Tx, _ *sqlcdb.Queries) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM departments WHERE code = 'SMUGGLED'`).Scan(&leaked)
	}); err != nil {
		t.Fatal(err)
	}
	if leaked != 0 {
		t.Fatal("cross-tenant insert landed in tenant A")
	}
}

// TestRLSDeniesWhenTenantUnset requires fail-closed when app.tenant_id is unset.
func TestRLSDeniesWhenTenantUnset(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	tenant := makeTenant(t, pool, "rls-unset-")

	if err := pool.WithTenant(ctx, tenant.ID, func(ctx context.Context, q *sqlcdb.Queries) error {
		_, err := q.CreateDepartment(ctx, sqlcdb.CreateDepartmentParams{
			TenantID: tenant.ID, Code: "MATH", Name: "Mathematics",
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	var visible int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM departments`).Scan(&visible); err != nil {
		t.Fatal(err)
	}
	if visible != 0 {
		t.Fatalf("a connection with no tenant scope saw %d rows; RLS must fail closed", visible)
	}
}

// TestWithTenantRejectsZeroTenant refuses scoping to the nil UUID.
func TestWithTenantRejectsZeroTenant(t *testing.T) {
	pool := testPool(t)
	err := pool.WithTenant(context.Background(), uuid.Nil, func(context.Context, *sqlcdb.Queries) error {
		t.Fatal("callback ran with an unset tenant")
		return nil
	})
	if err == nil {
		t.Fatal("expected WithTenant to reject the zero tenant id")
	}
}

func TestReservedAndMalformedSlugsRejected(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	for _, slug := range []string{"admin", "api", "www", "-leading", "trailing-", "Upper", "a", "has_underscore"} {
		_, err := pool.Platform().CreateTenant(ctx, sqlcdb.CreateTenantParams{Slug: slug, Name: "Nope"})
		if err == nil {
			t.Fatalf("slug %q should have been rejected by a constraint", slug)
		}
	}
}

func TestTenantSlugUniqueness(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	slug := "uniq-" + uuid.NewString()[:8]
	if _, err := pool.Platform().CreateTenant(ctx, sqlcdb.CreateTenantParams{Slug: slug, Name: "One"}); err != nil {
		t.Fatal(err)
	}
	_, err := pool.Platform().CreateTenant(ctx, sqlcdb.CreateTenantParams{Slug: slug, Name: "Two"})
	if err == nil {
		t.Fatal("expected unique slug violation")
	}
	if errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("unexpected error kind: %v", err)
	}
}
