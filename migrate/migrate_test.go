package migrate_test

import (
	"context"
	"testing"

	"github.com/sonalys/codemigrate/migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type customTransaction struct {
	getCurrentVersion func(ctx context.Context) (int64, error)
	setVersion        func(ctx context.Context, version int64) error
}

type customConnection[T migrate.Transaction] struct {
	transaction func(ctx context.Context, handler func(tx T) error) error
}

type customMigration struct {
	version int64
}

func (m customMigration) Up(ctx context.Context, tx customTransaction) error {
	return nil
}

func (m customMigration) Down(ctx context.Context, tx customTransaction) error {
	return nil
}

func (m customMigration) Version() int64 {
	return m.version
}

func (c customTransaction) GetCurrentVersion(ctx context.Context) (int64, error) {
	return c.getCurrentVersion(ctx)
}

func (c customTransaction) SetVersion(ctx context.Context, version int64) error {
	return c.setVersion(ctx, version)
}

func (c customConnection[T]) Transaction(ctx context.Context, handler func(tx T) error) error {
	return c.transaction(ctx, handler)
}

func Test_Migrator_Up(t *testing.T) {
	transaction := customTransaction{}

	conn := customConnection[customTransaction]{
		transaction: func(ctx context.Context, handler func(tx customTransaction) error) error {
			return handler(transaction)
		},
	}

	t.Run("error: migration conflict", func(t *testing.T) {
		migrator, err := migrate.New(conn,
			customMigration{version: 1},
			customMigration{version: 2},
			customMigration{version: 2},
		)
		require.ErrorIs(t, err, migrate.ErrDuplicateMigration)
		require.Nil(t, migrator)
	})

	t.Run("error: no migrations", func(t *testing.T) {
		ctx := t.Context()
		migrator, err := migrate.New(conn)
		require.NoError(t, err)
		require.NotNil(t, migrator)

		transaction.getCurrentVersion = func(ctx context.Context) (int64, error) {
			return 0, nil
		}

		err = migrator.Up(ctx, 1)
		require.ErrorIs(t, err, migrate.ErrNoMigrations)
	})

	t.Run("success: up-to-date", func(t *testing.T) {
		ctx := t.Context()
		migrator, err := migrate.New(conn)
		require.NoError(t, err)
		require.NotNil(t, migrator)

		transaction.getCurrentVersion = func(ctx context.Context) (int64, error) {
			return 3, nil
		}

		err = migrator.Up(ctx, 3)
		require.NoError(t, err)
	})

	t.Run("success: migrates one version up", func(t *testing.T) {
		ctx := t.Context()
		migrator, err := migrate.New(conn,
			customMigration{version: 1},
			customMigration{version: 3},
			customMigration{version: 4},
			customMigration{version: 5},
		)
		require.NoError(t, err)
		require.NotNil(t, migrator)

		count := 0
		transaction.getCurrentVersion = func(ctx context.Context) (int64, error) {
			return int64(3 + count), nil
		}

		transaction.setVersion = func(ctx context.Context, version int64) error {
			count++
			require.EqualValues(t, 4, version)
			return nil
		}

		err = migrator.Up(ctx, 4)
		require.NoError(t, err)
	})

	t.Run("success: migrates multiple versions up", func(t *testing.T) {
		ctx := t.Context()
		migrator, err := migrate.New(conn,
			customMigration{version: 1},
			customMigration{version: 2},
			customMigration{version: 3},
			customMigration{version: 4},
		)
		require.NoError(t, err)
		require.NotNil(t, migrator)

		count := 0

		transaction.getCurrentVersion = func(ctx context.Context) (int64, error) {
			return int64(1 + count), nil
		}

		transaction.setVersion = func(ctx context.Context, version int64) error {
			count++
			assert.EqualValues(t, 1+count, version)
			return nil
		}

		err = migrator.Up(ctx, 3)
		require.NoError(t, err)
	})

	t.Run("error: migration to target version not found", func(t *testing.T) {
		ctx := t.Context()
		migrator, err := migrate.New(conn,
			customMigration{version: 1},
		)
		require.NoError(t, err)
		require.NotNil(t, migrator)

		transaction.getCurrentVersion = func(ctx context.Context) (int64, error) {
			return 1, nil
		}

		err = migrator.Up(ctx, 2)
		require.ErrorIs(t, err, migrate.ErrMigrationNotFound)
	})

	t.Run("error: migrations skipped target version", func(t *testing.T) {
		ctx := t.Context()
		migrator, err := migrate.New(conn,
			customMigration{version: 1},
			customMigration{version: 3},
		)
		require.NoError(t, err)
		require.NotNil(t, migrator)

		transaction.getCurrentVersion = func(ctx context.Context) (int64, error) {
			return 1, nil
		}

		err = migrator.Up(ctx, 2)
		require.ErrorIs(t, err, migrate.ErrMigrationNotFound)
	})
}
