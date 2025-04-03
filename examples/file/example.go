package examples_test

import (
	"database/sql"
	"embed"
	"testing"

	"github.com/sonalys/codemigrate/database/postgres/pq/adapter"
	"github.com/sonalys/codemigrate/migrate"
	"github.com/stretchr/testify/require"
)

//go:embed migrations/*.sql
var fs embed.FS

func Test_Example_Code(t *testing.T) {
	ctx := t.Context()

	conn, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=postgres dbname=test sslmode=disable")
	require.NoError(t, err)

	v := adapter.From(conn)

	migration_0001, err := adapter.NewScriptMigrationFromFile[*sql.Tx](1, fs, "0001_init.up.sql", "0001_init.down.sql")
	require.NoError(t, err)

	migrator, err := migrate.New(v,
		migration_0001,
	)
	require.NoError(t, err)

	err = migrator.Up(ctx, migrate.Latest)
	require.NoError(t, err)

	err = migrator.Down(ctx, migrate.Oldest)
	require.NoError(t, err)
}
