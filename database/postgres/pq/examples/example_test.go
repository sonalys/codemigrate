package examples_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/sonalys/codemigrate/database/postgres/pq/adapter"
	"github.com/stretchr/testify/require"

	"github.com/sonalys/codemigrate/migrate"
)

type migration_0001 struct{}

func (m *migration_0001) Version() int64 {
	return 1
}

func (m *migration_0001) Up(ctx context.Context, tx *adapter.Versioner[*sql.Tx]) error {
	_, err := tx.Tx.Exec("CREATE TABLE IF NOT EXISTS test (id SERIAL PRIMARY KEY, name TEXT)")
	if err != nil {
		return err
	}
	return nil
}

func (m *migration_0001) Down(ctx context.Context, tx *adapter.Versioner[*sql.Tx]) error {
	_, err := tx.Tx.Exec("DROP TABLE IF EXISTS test")
	if err != nil {
		return err
	}
	return nil
}

func Test_Example(t *testing.T) {
	ctx := t.Context()

	conn, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=postgres dbname=test sslmode=disable")
	require.NoError(t, err)

	v := adapter.From(conn)

	migrator, err := migrate.New(v,
		&migration_0001{},
	)
	require.NoError(t, err)

	err = migrator.Up(ctx, migrate.Latest)
	require.NoError(t, err)

	err = migrator.Down(ctx, migrate.Oldest)
	require.NoError(t, err)
}
