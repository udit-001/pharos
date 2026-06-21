package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var migrationsFS embed.FS

// Up runs all pending migrations.
//
// IMPORTANT: goose.Provider.Close() calls sql.DB.Close() on the passed
// connection. Since the caller owns the DB handle, we never call
// provider.Close() here. The provider is left to be GC'd.
func Up(db *sql.DB) error {
	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		db,
		migrationsFS,
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	results, err := provider.Up(context.Background())
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	for _, r := range results {
		if r.Error != nil {
			return fmt.Errorf("migration %s failed: %w", r.Source.Path, r.Error)
		}
	}

	return nil
}

// Down rolls back the most recent migration.
func Down(db *sql.DB) error {
	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		db,
		migrationsFS,
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	result, err := provider.Down(context.Background())
	if err != nil {
		return fmt.Errorf("rollback migration: %w", err)
	}

	if result != nil && result.Error != nil {
		return fmt.Errorf("migration %s failed: %w", result.Source.Path, result.Error)
	}

	return nil
}

// Status returns the current migration status.
func Status(db *sql.DB) ([]*goose.MigrationStatus, error) {
	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		db,
		migrationsFS,
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return nil, fmt.Errorf("create migration provider: %w", err)
	}

	return provider.Status(context.Background())
}

// UpTo runs migrations up to a specific version.
func UpTo(db *sql.DB, version int64) error {
	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		db,
		migrationsFS,
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	results, err := provider.UpTo(context.Background(), version)
	if err != nil {
		return fmt.Errorf("run migrations up to %d: %w", version, err)
	}

	for _, r := range results {
		if r.Error != nil {
			return fmt.Errorf("migration %s failed: %w", r.Source.Path, r.Error)
		}
	}

	return nil
}

// DownTo rolls back migrations to a specific version.
func DownTo(db *sql.DB, version int64) error {
	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		db,
		migrationsFS,
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	results, err := provider.DownTo(context.Background(), version)
	if err != nil {
		return fmt.Errorf("rollback migrations to %d: %w", version, err)
	}

	for _, r := range results {
		if r.Error != nil {
			return fmt.Errorf("migration %s failed: %w", r.Source.Path, r.Error)
		}
	}

	return nil
}
