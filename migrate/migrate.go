package migrate

import (
	"context"
	"fmt"
	"sort"
)

type (
	Transaction interface {
		GetCurrentVersion(ctx context.Context) (int64, error)
		SetVersion(ctx context.Context, version int64) error
	}

	Transactioner[T Transaction] interface {
		Transaction(ctx context.Context, handler func(tx T) error) error
	}

	Migration[T Transaction] interface {
		Up(ctx context.Context, tx T) error
		Down(ctx context.Context, tx T) error
		Version() int64
	}

	Migrator[T Transaction] struct {
		conn       Transactioner[T]
		migrations []Migration[T]
	}

	StringError string
)

const (
	// ErrNoMigrations is returned when no migrations are applied.
	ErrNoMigrations = StringError("no migrations applied")
	// ErrMigrationNotFound is returned when a migration is not found.
	ErrMigrationNotFound = StringError("migration not found")
	// ErrDuplicateMigration is returned when a duplicate migration version is found.
	ErrDuplicateMigration = StringError("duplicate migration version")
)

func (e StringError) Error() string {
	return string(e)
}

func New[T Transaction](conn Transactioner[T], migrations ...Migration[T]) (*Migrator[T], error) {
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version() < migrations[j].Version()
	})

	for i := 0; i < len(migrations)-1; i++ {
		for j := i + 1; j < len(migrations); j++ {
			if migrations[i].Version() == migrations[j].Version() {
				return nil, fmt.Errorf("could not apply migration version %d: %w", migrations[i].Version(), ErrDuplicateMigration)
			}
		}
	}

	return &Migrator[T]{
		conn:       conn,
		migrations: migrations,
	}, nil
}

func (m Migrator[T]) findNextMigration(currentVersion int64) (int64, Migration[T]) {
	for _, migration := range m.migrations {
		if migration.Version() > currentVersion {
			return migration.Version(), migration
		}
	}
	return -1, nil
}

func (m Migrator[T]) Up(ctx context.Context, targetVersion int64) error {
	db := m.conn
	var lastAppliedVersion int64

	for runAgain := true; runAgain; {
		runAgain = false

		err := db.Transaction(ctx, func(tx T) error {
			currentVersion, err := tx.GetCurrentVersion(ctx)
			if err != nil {
				return fmt.Errorf("getting current version: %w", err)
			}

			if currentVersion == targetVersion {
				lastAppliedVersion = currentVersion
				return nil
			}

			nextVersion, migration := m.findNextMigration(currentVersion)
			if nextVersion == -1 {
				if currentVersion < targetVersion {
					return fmt.Errorf("migrating to target version %d: %w", targetVersion, ErrMigrationNotFound)
				}
				return nil
			}

			if nextVersion > targetVersion {
				return fmt.Errorf("migrating to target version %d: %w", targetVersion, ErrMigrationNotFound)
			}

			err = migration.Up(ctx, tx)
			if err != nil {
				return fmt.Errorf("applying migration %d: %w", nextVersion, err)
			}

			err = tx.SetVersion(ctx, nextVersion)
			if err != nil {
				return fmt.Errorf("updating version: %w", err)
			}

			lastAppliedVersion = nextVersion
			runAgain = true
			return nil
		})

		if err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	if lastAppliedVersion == 0 {
		return ErrNoMigrations
	}

	return nil
}
