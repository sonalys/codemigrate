package examples_test

import (
	"bytes"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sonalys/codemigrate/database/postgres/pgx/adapter"
	"github.com/sonalys/codemigrate/migrate"
	"github.com/stretchr/testify/require"
)

func Test_Example_File(t *testing.T) {
	ctx := t.Context()

	conn, err := pgx.Connect(ctx, "host=localhost port=5432 user=postgres password=postgres dbname=test sslmode=disable")
	require.NoError(t, err)

	v := adapter.From(conn)

	upScript := bytes.NewBufferString(`CREATE TABLE IF NOT EXISTS test (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL
	);`)

	migration, err := adapter.NewScriptMigrationFromReader(1, upScript, nil)
	require.NoError(t, err)

	migrator, err := migrate.New(v,
		migration,
	)
	require.NoError(t, err)

	err = migrator.Up(ctx, migrate.Latest)
	require.NoError(t, err)

	err = migrator.Down(ctx, migrate.Oldest)
	require.NoError(t, err)
}
