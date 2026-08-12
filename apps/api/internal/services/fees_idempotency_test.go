package services_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type payFixture struct {
	tenantID  uuid.UUID
	studentID uuid.UUID
	feeHeadID uuid.UUID
}

func seedPayFixture(t *testing.T, pool *db.Pool, prefix string) payFixture {
	t.Helper()
	ctx := context.Background()
	tenant, err := pool.Platform().CreateTenant(ctx, sqlcdb.CreateTenantParams{
		Slug: prefix + uuid.NewString()[:8],
		Name: "Pay Tenant",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Platform().UpdateTenantStatus(ctx, sqlcdb.UpdateTenantStatusParams{
		ID: tenant.ID, Status: "active",
	}); err != nil {
		t.Fatal(err)
	}

	f := payFixture{tenantID: tenant.ID}
	err = pool.WithTenant(ctx, tenant.ID, func(ctx context.Context, q *sqlcdb.Queries) error {
		hash, err := auth.HashPassword("correct horse battery")
		if err != nil {
			return err
		}
		u, err := q.CreateUser(ctx, sqlcdb.CreateUserParams{
			TenantID:     tenant.ID,
			Email:        uuid.NewString() + "@example.com",
			PasswordHash: hash,
			Role:         "student",
			FullName:     "Student",
		})
		if err != nil {
			return err
		}
		st, err := q.CreateStudent(ctx, sqlcdb.CreateStudentParams{
			TenantID:   tenant.ID,
			UserID:     u.ID,
			RollNumber: uuid.NewString()[:8],
			Program:    "BTech",
			BatchYear:  2024,
		})
		if err != nil {
			return err
		}
		f.studentID = st.ID
		fh, err := q.CreateFeeHead(ctx, sqlcdb.CreateFeeHeadParams{
			TenantID:           tenant.ID,
			Name:               "Tuition",
			Amount:             services.NumericFromFloat(1000),
			LateFeeAmount:      services.NumericFromFloat(0),
			ApplicablePrograms: []string{},
		})
		if err != nil {
			return err
		}
		f.feeHeadID = fh.ID
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func servicesPool(t *testing.T) *db.Pool {
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

func TestPaymentIdempotencySequential(t *testing.T) {
	pool := servicesPool(t)
	f := seedPayFixture(t, pool, "pay-")
	fees := services.NewFeesService(pool)

	in := services.CreatePaymentInput{
		StudentID:      f.studentID,
		FeeHeadID:      f.feeHeadID,
		IdempotencyKey: "idem-" + uuid.NewString(),
	}
	p1, err := fees.CreatePayment(context.Background(), f.tenantID, in)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := fees.CreatePayment(context.Background(), f.tenantID, in)
	if err != nil {
		t.Fatal(err)
	}
	if p1.ID != p2.ID {
		t.Fatalf("idempotency failed: %s vs %s", p1.ID, p2.ID)
	}
}

// The same request fired concurrently — a double-clicked pay button, or a
// client retrying on a timeout. Exactly one row may exist afterwards, and every
// caller must get that row back rather than an error.
func TestPaymentIdempotencyConcurrent(t *testing.T) {
	pool := servicesPool(t)
	f := seedPayFixture(t, pool, "payc-")
	fees := services.NewFeesService(pool)

	const attempts = 12
	in := services.CreatePaymentInput{
		StudentID:      f.studentID,
		FeeHeadID:      f.feeHeadID,
		IdempotencyKey: "idem-" + uuid.NewString(),
	}

	ids := make([]uuid.UUID, attempts)
	errs := make([]error, attempts)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			p, err := fees.CreatePayment(context.Background(), f.tenantID, in)
			ids[i], errs[i] = p.ID, err
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("attempt %d failed instead of replaying the existing payment: %v", i, err)
		}
		if ids[i] != ids[0] {
			t.Fatalf("attempt %d returned payment %s, attempt 0 returned %s", i, ids[i], ids[0])
		}
	}

	var count int
	if err := pool.WithTenantTx(context.Background(), f.tenantID, func(ctx context.Context, tx pgx.Tx, _ *sqlcdb.Queries) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM fee_payments WHERE idempotency_key = $1`, in.IdempotencyKey).Scan(&count)
	}); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 payment row, found %d", count)
	}
}

// Idempotency keys are unique per tenant, not per student. Replaying another
// student's key must not hand back their payment record.
func TestPaymentIdempotencyKeyReuseWithDifferentPayloadRejected(t *testing.T) {
	pool := servicesPool(t)
	ctx := context.Background()
	f := seedPayFixture(t, pool, "payr-")
	fees := services.NewFeesService(pool)

	key := "idem-" + uuid.NewString()
	if _, err := fees.CreatePayment(ctx, f.tenantID, services.CreatePaymentInput{
		StudentID: f.studentID, FeeHeadID: f.feeHeadID, IdempotencyKey: key,
	}); err != nil {
		t.Fatal(err)
	}

	_, err := fees.CreatePayment(ctx, f.tenantID, services.CreatePaymentInput{
		StudentID: uuid.New(), FeeHeadID: f.feeHeadID, IdempotencyKey: key,
	})
	if !errors.Is(err, services.ErrConflict) {
		t.Fatalf("reusing a key with a different student got %v, want ErrConflict", err)
	}
}

// A payment must not be creatable against another tenant's fee head, even with
// a valid id for it.
func TestPaymentRejectsCrossTenantFeeHead(t *testing.T) {
	pool := servicesPool(t)
	ctx := context.Background()
	a := seedPayFixture(t, pool, "payx-a-")
	b := seedPayFixture(t, pool, "payx-b-")
	fees := services.NewFeesService(pool)

	_, err := fees.CreatePayment(ctx, a.tenantID, services.CreatePaymentInput{
		StudentID:      a.studentID,
		FeeHeadID:      b.feeHeadID,
		IdempotencyKey: "idem-" + uuid.NewString(),
	})
	if !errors.Is(err, services.ErrNotFound) {
		t.Fatalf("cross-tenant fee head got %v, want ErrNotFound", err)
	}
}

func TestListDuesRejectsCrossTenantStudent(t *testing.T) {
	pool := servicesPool(t)
	ctx := context.Background()
	a := seedPayFixture(t, pool, "dues-a-")
	b := seedPayFixture(t, pool, "dues-b-")
	fees := services.NewFeesService(pool)

	if _, err := fees.ListDues(ctx, a.tenantID, b.studentID); !errors.Is(err, services.ErrNotFound) {
		t.Fatalf("cross-tenant student got %v, want ErrNotFound", err)
	}
	if _, err := fees.ListDues(ctx, a.tenantID, a.studentID); err != nil {
		t.Fatalf("own student should resolve: %v", err)
	}
}
