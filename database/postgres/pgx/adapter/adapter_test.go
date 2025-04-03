package adapter_test

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sonalys/codemigrate/database/postgres/pgx/adapter"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgres_Transaction(t *testing.T) {
	ctx := t.Context()

	pgContainer, err := postgres.Run(ctx, "postgres:16",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	defer func() {
		err := pgContainer.Terminate(ctx)
		require.NoError(t, err)
	}()

	connStr, err := pgContainer.ConnectionString(ctx)
	require.NoError(t, err)

	conn, err := pgx.Connect(ctx, connStr)
	require.NoError(t, err)
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()

	pg := adapter.From(conn, adapter.WithTableName("test_migrations"))

	err = pg.Transaction(ctx, func(tx *adapter.Versioner) error {
		version, err := tx.GetCurrentVersion(ctx)
		require.NoError(t, err)
		require.EqualValues(t, 0, version)

		err = tx.SetVersion(ctx, 1)
		require.NoError(t, err)

		version, err = tx.GetCurrentVersion(ctx)
		require.NoError(t, err)
		require.EqualValues(t, 1, version)

		return nil
	})
	require.NoError(t, err)
}
