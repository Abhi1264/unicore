package services_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/google/uuid"
)

func TestEnrollmentSeatCapNoOversell(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	ensureMigrated(t)
	pool, err := db.NewPool(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	q := pool.Platform()
	tenant, err := q.CreateTenant(ctx, sqlcdb.CreateTenantParams{
		Slug: "enr-" + uuid.NewString()[:8],
		Name: "Enroll Tenant",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = q.UpdateTenantStatus(ctx, sqlcdb.UpdateTenantStatusParams{ID: tenant.ID, Status: "active"})

	var courseID uuid.UUID
	studentIDs := make([]uuid.UUID, 20)

	err = pool.WithTenant(ctx, tenant.ID, func(ctx context.Context, tq *sqlcdb.Queries) error {
		course, err := tq.CreateCourse(ctx, sqlcdb.CreateCourseParams{
			TenantID: tenant.ID,
			Code:     "CS101",
			Name:     "Intro",
			Credits:  services.NumericFromFloat(3),
			SeatCap:  5,
		})
		if err != nil {
			return err
		}
		courseID = course.ID

		_, err = tq.CreateRegistrationWindow(ctx, sqlcdb.CreateRegistrationWindowParams{
			TenantID: tenant.ID,
			Name:     "Spring",
			Semester: "2026S1",
			OpensAt:  time.Now().Add(-time.Hour),
			ClosesAt: time.Now().Add(time.Hour),
		})
		if err != nil {
			return err
		}

		for i := 0; i < 20; i++ {
			hash, _ := auth.HashPassword("password123")
			u, err := tq.CreateUser(ctx, sqlcdb.CreateUserParams{
				TenantID:     tenant.ID,
				Email:        uuid.NewString() + "@ex.com",
				PasswordHash: hash,
				Role:         "student",
				FullName:     "S",
			})
			if err != nil {
				return err
			}
			st, err := tq.CreateStudent(ctx, sqlcdb.CreateStudentParams{
				TenantID:   tenant.ID,
				UserID:     u.ID,
				RollNumber: uuid.NewString()[:8],
				Program:    "BTech",
				BatchYear:  2024,
			})
			if err != nil {
				return err
			}
			studentIDs[i] = st.ID
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	academic := services.NewAcademicService(pool)
	var success atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(sid uuid.UUID) {
			defer wg.Done()
			_, err := academic.EnrollStudent(ctx, tenant.ID, services.EnrollStudentInput{
				StudentID:      sid,
				CourseID:       courseID,
				Semester:       "2026S1",
				IdempotencyKey: uuid.NewString(),
			})
			if err == nil {
				success.Add(1)
			}
		}(studentIDs[i])
	}
	wg.Wait()

	if success.Load() != 5 {
		t.Fatalf("expected exactly 5 enrollments, got %d", success.Load())
	}
}
