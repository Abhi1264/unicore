package db_test

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/Abhi1264/unicore/api/internal/db"
)

var migrateOnce sync.Once

func ensureMigrated(t *testing.T) {
	t.Helper()
	migrateOnce.Do(func() {
		url := os.Getenv("DATABASE_MIGRATE_URL")
		if url == "" {
			url = os.Getenv("DATABASE_URL")
		}
		if url == "" {
			return
		}
		_, file, _, _ := runtime.Caller(0)
		path := filepath.Join(filepath.Dir(file), "migrations")
		if err := db.RunMigrations(url, path); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	})
}
