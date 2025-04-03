package migrate

import (
	"context"
	"fmt"
)

type (
	migrator[T Versioner] struct {
		conn       Database[T]
		migrations []Migration[T]
	}
)

func (m migrator[T]) handler(ctx context.Context, targetVersion int64, h func(currentVersion int64, tx T) (bool, int64, error)) error {
	db := m.conn
	lastAppliedVersion := int64(-1)

	if len(m.migrations) == 0 {
		return ErrNoMigrations
	}

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

			var nextVersion int64
			runAgain, nextVersion, err = h(currentVersion, tx)
			if err != nil {
				return fmt.Errorf("running handler: %w", err)
			}

			err = tx.SetVersion(ctx, nextVersion)
			if err != nil {
				return fmt.Errorf("setting new version: %w", err)
			}

			lastAppliedVersion = nextVersion
			return nil
		})
		if err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	if lastAppliedVersion == -1 {
		return ErrNoMigrations
	}

	return nil
}

func (m migrator[T]) Up(ctx context.Context, targetVersion int64) error {
	if len(m.migrations) == 0 {
		return ErrNoMigrations
	}

	if targetVersion == Latest {
		targetVersion = m.migrations[len(m.migrations)-1].Version()
	}

	handle := func(currentVersion int64, tx T) (bool, int64, error) {
		nextVersion, migration := m.findNextMigration(currentVersion)
		if nextVersion == -1 {
			if currentVersion < targetVersion {
				return false, currentVersion, ErrMigrationNotFound
			}
			return false, currentVersion, nil
		}
		if nextVersion > targetVersion {
			return false, currentVersion, ErrMigrationNotFound
		}

		err := migration.Up(ctx, tx)
		if err != nil {
			return false, currentVersion, fmt.Errorf("applying migration %d: %w", nextVersion, err)
		}

		return nextVersion < targetVersion, nextVersion, nil
	}

	if err := m.handler(ctx, targetVersion, handle); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	return nil
}

func (m migrator[T]) Down(ctx context.Context, targetVersion int64) error {
	if len(m.migrations) == 0 {
		return ErrNoMigrations
	}

	if targetVersion == Oldest {
		targetVersion = m.migrations[0].Version()
	}

	handle := func(currentVersion int64, tx T) (bool, int64, error) {
		nextVersion, migration := m.findPrevMigration(currentVersion)
		if nextVersion == -1 {
			if currentVersion > targetVersion {
				return false, currentVersion, ErrMigrationNotFound
			}
			return false, currentVersion, nil
		}

		if nextVersion < targetVersion {
			return false, currentVersion, ErrMigrationNotFound
		}

		if err := migration.Down(ctx, tx); err != nil {
			return false, currentVersion, fmt.Errorf("applying migration %d: %w", nextVersion, err)
		}

		return nextVersion > targetVersion, nextVersion, nil
	}

	if err := m.handler(ctx, targetVersion, handle); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	return nil
}

func validateMigrations[T Versioner](migrations ...Migration[T]) error {
	if len(migrations) == 0 {
		return ErrNoMigrations
	}

	for i := 0; i < len(migrations)-1; i++ {
		for j := i + 1; j < len(migrations); j++ {
			if migrations[i].Version() == migrations[j].Version() {
				return fmt.Errorf("could not apply migration version %d: %w", migrations[i].Version(), ErrDuplicateMigration)
			}
		}
	}
	return nil
}

func (m migrator[T]) findNextMigration(currentVersion int64) (int64, Migration[T]) {
	for _, migration := range m.migrations {
		if migration.Version() > currentVersion {
			return migration.Version(), migration
		}
	}
	return -1, nil
}

func (m migrator[T]) findPrevMigration(currentVersion int64) (int64, Migration[T]) {
	for i := len(m.migrations) - 1; i >= 0; i-- {
		if m.migrations[i].Version() < currentVersion {
			return m.migrations[i].Version(), m.migrations[i]
		}
	}
	return -1, nil
}
