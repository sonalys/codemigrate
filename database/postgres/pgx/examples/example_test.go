package examples_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sonalys/codemigrate/database/postgres/pgx/versioner"
	"github.com/stretchr/testify/require"

	"github.com/sonalys/codemigrate/migrate"
)

type migration_0001 struct{}

func (m *migration_0001) Version() int64 {
	return 1
}

func (m *migration_0001) Up(ctx context.Context, tx *versioner.Versioner) error {
	_, err := tx.Exec(ctx, "CREATE TABLE IF NOT EXISTS test (id SERIAL PRIMARY KEY, name TEXT)")
	if err != nil {
		return err
	}
	return nil
}

func (m *migration_0001) Down(ctx context.Context, tx *versioner.Versioner) error {
	_, err := tx.Exec(ctx, "DROP TABLE IF EXISTS test")
	if err != nil {
		return err
	}
	return nil
}

func Test_Example(t *testing.T) {
	ctx := t.Context()

	conn, err := pgx.Connect(ctx, "host=localhost port=5432 user=postgres password=postgres dbname=test sslmode=disable")
	require.NoError(t, err)

	v := versioner.From(conn)

	migrator, err := migrate.New(v,
		&migration_0001{},
	)
	require.NoError(t, err)

	err = migrator.Up(ctx)
	require.NoError(t, err)

}
