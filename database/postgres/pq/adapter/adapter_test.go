package adapter_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/sonalys/codemigrate/database/postgres/pq/adapter"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	_ "github.com/lib/pq"
)

func TestPostgres_Transaction(t *testing.T) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:16",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx)

	connStr, err := pgContainer.ConnectionString(ctx)
	require.NoError(t, err)

	connStr += " sslmode=disable"

	conn, err := sql.Open("postgres", connStr)
	require.NoError(t, err)
	defer conn.Close()

	pg := adapter.From(conn, adapter.WithTableName("test_migrations"))

	err = pg.Transaction(ctx, func(tx *adapter.Versioner[*sql.Tx]) error {
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
