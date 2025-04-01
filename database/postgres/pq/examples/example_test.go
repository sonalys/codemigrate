package examples_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/sonalys/codemigrate/database/postgres/pq/versioner"
	"github.com/stretchr/testify/require"

	"github.com/sonalys/codemigrate/migrate"
)

type migration struct{}

func (m *migration) Up(ctx context.Context, tx *versioner.Versioner[*sql.Tx]) error {
	_, err := tx.Transaction.Exec("CREATE TABLE IF NOT EXISTS test (id SERIAL PRIMARY KEY, name TEXT)")
	if err != nil {
		return err
	}
	return nil
}

func (m *migration) Down(ctx context.Context, tx *versioner.Versioner[*sql.Tx]) error {
	_, err := tx.Transaction.Exec("DROP TABLE IF EXISTS test")
	if err != nil {
		return err
	}
	return nil
}

func Test_Example(t *testing.T) {
	ctx := t.Context()

	conn, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=postgres dbname=test sslmode=disable")
	require.NoError(t, err)

	v := versioner.From(conn)

	controller := migrate.NewMigrationController(map[int]migrate.Migration[*versioner.Versioner[*sql.Tx]]{
		1: &migration{},
	})

	targetVersion := 1

	err = migrate.Up(ctx, v, targetVersion, controller)
	require.NoError(t, err)
}
