package db

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(databaseURL, migrationsPath string) error {
	// migrate pgx driver expects postgres:// or pgx5://
	url := databaseURL
	m, err := migrate.New("file://"+migrationsPath, toMigrateURL(url))
	if err != nil {
		return fmt.Errorf("migrate new: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func toMigrateURL(databaseURL string) string {
	// Use pgx5 scheme for migrate v4 pgx/v5 driver
	if len(databaseURL) > 11 && databaseURL[:11] == "postgres://" {
		return "pgx5://" + databaseURL[11:]
	}
	if len(databaseURL) > 13 && databaseURL[:13] == "postgresql://" {
		return "pgx5://" + databaseURL[13:]
	}
	return databaseURL
}
