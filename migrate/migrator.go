package migrate

import (
	"context"
	"fmt"
	"sort"
)

type (
	// Versioner abstracts a component capable of reading and writing version information.
	// It's always used inside a transaction.
	// It's implemented by a versioner. Example: github.com/sonalys/codemigrate/databases/postgres/pgx/adapter.
	Versioner interface {
		GetCurrentVersion(ctx context.Context) (int64, error)
		SetVersion(ctx context.Context, version int64) error
	}

	// Database abstracts a database wrapper that can be used to perform transactions.
	// It's implemented by a database. Example: github.com/sonalys/codemigrate/databases/postgres/pgx/adapter.
	Database[T Versioner] interface {
		Transaction(ctx context.Context, handler func(tx T) error) error
	}

	// Migration abstracts each migration that can be applied to the database.
	// You can implement this interface to create your own migrations.
	Migration[T Versioner] interface {
		// Up applies the version bump to the database.
		Up(ctx context.Context, tx T) error
		// Down reverts the version bump to the database.
		Down(ctx context.Context, tx T) error
		// Version returns the version of the migration.
		// It should be unique and positive. It doesn't have to be sequential.
		// Example: 1, Unix timestamp, etc.
		Version() int64
	}

	// Migrator abstracts the migration process.
	// It can be used to apply or revert migrations.
	// It's initialized by the New function.
	Migrator interface {
		// Up applies the migrations to the database.
		// If it's running by the first time, it will apply all migrations.
		// If no migrations were applied, it will return ErrNoMigrations.
		// TargetVersion should be greater than the current version, or it will return ErrNoMigrations.
		// There must be a migration for the target version, or it will return ErrMigrationNotFound.
		// You can use migrate.Latest to apply all migrations.
		Up(ctx context.Context, targetVersion int64) error
		// Down reverts the migrations to the database.
		// If no migrations were applied, it will return ErrNoMigrations.
		// TargetVersion should be less than the current version, or it will return ErrNoMigrations.
		// There must be a migration for the target version, or it will return ErrMigrationNotFound.
		// You can use migrate.Oldest to revert all migrations.
		Down(ctx context.Context, targetVersion int64) error
	}
)

const (
	// Latest is a special value that can be used to apply all migrations.
	Latest int64 = -1
	// Oldest is a special value that can be used to revert all migrations.
	Oldest int64 = -2
)

// New creates a new migrator.
// It takes a database connection and a list of migrations.
// It returns a migrator that can be used to apply or revert migrations.
// It sorts the migrations by version and validates them.
// At least one migration must be provided.
func New[T Versioner](conn Database[T], migrations ...Migration[T]) (Migrator, error) {
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version() < migrations[j].Version()
	})

	if err := validateMigrations(migrations...); err != nil {
		return nil, fmt.Errorf("validating migrations: %w", err)
	}

	return &migrator[T]{
		conn:       conn,
		migrations: migrations,
	}, nil
}
