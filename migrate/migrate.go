package migrate

import (
	"context"
	"fmt"
)

type (
	Database[V Versioner] interface {
		Transaction(ctx context.Context, handler func(versioner V) error) error
	}

	Versioner interface {
		GetCurrentVersion(ctx context.Context) (int, error)
		SetVersion(ctx context.Context, version int) error
	}

	Migration[V Versioner] interface {
		Up(ctx context.Context, versioner V) error
		Down(ctx context.Context, versioner V) error
	}

	MigrationController[V Versioner] struct {
		migrations map[int]Migration[V]
	}
)

func Up[T Versioner](ctx context.Context, db Database[T], targetVersion int, controller *MigrationController[T]) error {
	for runAgain := true; runAgain; {
		runAgain = false

		err := db.Transaction(ctx, func(tx T) error {
			currentVersion, err := tx.GetCurrentVersion(ctx)
			if err != nil {
				return fmt.Errorf("getting current version: %w", err)
			}

			if currentVersion >= targetVersion {
				return nil
			}

			nextVersion := currentVersion + 1

			migration, err := controller.GetMigration(nextVersion)
			if err != nil {
				return fmt.Errorf("getting migration %d: %w", nextVersion, err)
			}

			err = migration.Up(ctx, tx)
			if err != nil {
				return fmt.Errorf("applying migration %d: %w", nextVersion, err)
			}

			err = tx.SetVersion(ctx, nextVersion)
			if err != nil {
				return fmt.Errorf("updating version: %w", err)
			}

			runAgain = nextVersion < targetVersion
			return nil
		})

		if err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

func NewMigrationController[T Versioner](from map[int]Migration[T]) *MigrationController[T] {
	return &MigrationController[T]{migrations: from}
}

func (c *MigrationController[T]) GetMigration(version int) (Migration[T], error) {
	migration, ok := c.migrations[version]
	if !ok {
		return nil, fmt.Errorf("migration %d not found", version)
	}
	return migration, nil
}
