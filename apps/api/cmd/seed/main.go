package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/Abhi1264/unicore/api/internal/db"
)

func main() {
	ctx := context.Background()
	cfg := loadSeedConfig()

	migrateURL := env("DATABASE_MIGRATE_URL", os.Getenv("DATABASE_URL"))
	if migrateURL == "" {
		log.Fatal("DATABASE_MIGRATE_URL or DATABASE_URL is required")
	}
	if err := db.RunMigrations(migrateURL, env("MIGRATIONS_PATH", "internal/db/migrations")); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, migrateURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	start := time.Now()
	q := pool.Platform()

	platform, err := ensureTenant(ctx, q, "platform", "Unicore Platform", "active")
	if err != nil {
		log.Fatal(err)
	}
	if err := seedSuperadmin(ctx, pool, platform.ID); err != nil {
		log.Fatal(err)
	}

	tenants := []struct{ slug, name string }{
		{"bitmesra", "BIT Mesra"},
		{"demo2", "Demo Institute"},
	}
	for _, t := range tenants {
		ten, err := ensureTenant(ctx, q, t.slug, t.name, "active")
		if err != nil {
			log.Fatal(err)
		}
		stats, err := seedTenant(ctx, pool, ten, cfg)
		if err != nil {
			log.Fatalf("seed %s: %v", t.slug, err)
		}
		fmt.Printf(
			"seeded %s · students=%d faculty=%d courses=%d enrollments=%d results=%d attendance=%d payments=%d announcements=%d\n",
			t.slug, stats.Students, stats.Faculty, stats.Courses, stats.Enrollments,
			stats.Results, stats.Attendance, stats.Payments, stats.Announcements,
		)
	}

	fmt.Printf("\ndone in %s\n", time.Since(start).Round(time.Millisecond))
	fmt.Println("passwords: Unicore-<role>-2026!  (override prefix with SEED_PASSWORD_PREFIX)")
	fmt.Println("hosts:     http://bitmesra.localhost:3000  http://demo2.localhost:3000  http://app.localhost:3000")
	fmt.Println("scale:     SEED_STUDENTS / SEED_FACULTY / SEED_ATTENDANCE_STUDENTS / SEED_ATTENDANCE_SESSIONS")
}

type seedConfig struct {
	Students           int
	Faculty            int
	AttendanceStudents int
	AttendanceSessions int
}

func loadSeedConfig() seedConfig {
	return seedConfig{
		Students:           envInt("SEED_STUDENTS", 200),
		Faculty:            envInt("SEED_FACULTY", 8),
		AttendanceStudents: envInt("SEED_ATTENDANCE_STUDENTS", 80),
		AttendanceSessions: envInt("SEED_ATTENDANCE_SESSIONS", 12),
	}
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func envInt(k string, d int) int {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return d
	}
	return n
}
