package examples_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sonalys/codemigrate/database/postgres/pgx/adapter"
	"github.com/sonalys/codemigrate/migrate"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type OldSchema struct {
	Version int    `json:"version"`
	Name    string `json:"name"`
	Value   int    `json:"value"`
}

type NewSchema struct {
	Version     int    `json:"version"`
	Description string `json:"description"`
	Value       int    `json:"value"`
}

type migration_0001 struct{}

func (m *migration_0001) Version() int64 {
	return 1
}

func (m *migration_0001) Up(ctx context.Context, tx *adapter.Versioner) error {
	_, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test (
			id SERIAL PRIMARY KEY,
			name TEXT,
			data JSONB NOT NULL DEFAULT '{}'::jsonb
		)
	`)
	if err != nil {
		return err
	}

	// Initialize data with SchemaA
	initialData := OldSchema{
		Version: 1,
		Name:    "example",
		Value:   123,
	}
	dataJSON, err := json.Marshal(initialData)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO test (name, data) VALUES
		('example', $1)
	`, dataJSON)
	return err
}

func (m *migration_0001) Down(ctx context.Context, tx *adapter.Versioner) error {
	_, err := tx.Exec(ctx, "DROP TABLE IF EXISTS test")
	return err
}

type migration_0002 struct{}

func (m *migration_0002) Version() int64 {
	return 2
}

func (m *migration_0002) Up(ctx context.Context, tx *adapter.Versioner) error {
	const pageSize = 100
	var lastID int

	type rowData struct {
		id   int
		data []byte
	}

	for {
		rows, err := tx.Query(ctx, `
			SELECT id, data FROM test
			WHERE id > $1
			ORDER BY id ASC
			LIMIT $2
		`, lastID, pageSize)
		if err != nil {
			return err
		}

		// Store all rows data before closing
		var processedRows []rowData
		for rows.Next() {
			var rd rowData
			if err := rows.Scan(&rd.id, &rd.data); err != nil {
				rows.Close()
				return err
			}
			processedRows = append(processedRows, rd)
			lastID = rd.id
		}
		rows.Close()

		if len(processedRows) == 0 {
			break
		}

		// Process stored rows
		var updateParams []interface{}
		var cases []string
		paramCount := 1

		for _, rd := range processedRows {
			var oldSchema OldSchema
			if err := json.Unmarshal(rd.data, &oldSchema); err != nil {
				return err
			}

			if oldSchema.Version != 1 {
				continue
			}

			newSchema := NewSchema{
				Version:     2,
				Description: "Updated description",
				Value:       oldSchema.Value,
			}

			updatedData, err := json.Marshal(newSchema)
			if err != nil {
				return err
			}

			cases = append(cases, fmt.Sprintf("WHEN id = $%d THEN $%d", paramCount, paramCount+1))
			updateParams = append(updateParams, rd.id, updatedData)
			paramCount += 2
		}

		if len(cases) > 0 {
			query := fmt.Sprintf(`
				UPDATE test SET data = CASE %s ELSE data END
				WHERE id IN (SELECT unnest($%d::int[]))
			`, strings.Join(cases, " "), paramCount)

			// Create array of IDs for the WHERE clause
			var ids []int
			for i := 0; i < len(updateParams); i += 2 {
				ids = append(ids, updateParams[i].(int))
			}
			updateParams = append(updateParams, ids)

			_, err = tx.Exec(ctx, query, updateParams...)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *migration_0002) Down(ctx context.Context, tx *adapter.Versioner) error {
	const pageSize = 100
	var lastID int

	type rowData struct {
		id   int
		data []byte
	}

	for {
		rows, err := tx.Query(ctx, `
			SELECT id, data FROM test
			WHERE id > $1
			ORDER BY id ASC
			LIMIT $2
		`, lastID, pageSize)
		if err != nil {
			return err
		}

		// Store all rows data before closing
		var processedRows []rowData
		for rows.Next() {
			var rd rowData
			if err := rows.Scan(&rd.id, &rd.data); err != nil {
				rows.Close()
				return err
			}
			processedRows = append(processedRows, rd)
			lastID = rd.id
		}
		rows.Close()

		if len(processedRows) == 0 {
			break
		}

		// Process stored rows
		var updateParams []interface{}
		var cases []string
		paramCount := 1

		for _, rd := range processedRows {
			var newSchema NewSchema
			if err := json.Unmarshal(rd.data, &newSchema); err != nil {
				return err
			}

			if newSchema.Version != 2 {
				continue
			}

			oldSchema := OldSchema{
				Version: 1,
				Name:    "example",
				Value:   newSchema.Value,
			}

			originalData, err := json.Marshal(oldSchema)
			if err != nil {
				return err
			}

			cases = append(cases, fmt.Sprintf("WHEN id = $%d THEN $%d", paramCount, paramCount+1))
			updateParams = append(updateParams, rd.id, originalData)
			paramCount += 2
		}

		if len(cases) > 0 {
			query := fmt.Sprintf(`
				UPDATE test SET data = CASE %s ELSE data END
				WHERE id IN (SELECT unnest($%d::int[]))
			`, strings.Join(cases, " "), paramCount)

			// Create array of IDs for the WHERE clause
			var ids []int
			for i := 0; i < len(updateParams); i += 2 {
				ids = append(ids, updateParams[i].(int))
			}
			updateParams = append(updateParams, ids)

			_, err = tx.Exec(ctx, query, updateParams...)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func Test_Example_Code(t *testing.T) {
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

	conn, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	v := adapter.From(conn)

	migrator, err := migrate.New(v,
		&migration_0001{},
		&migration_0002{},
	)
	require.NoError(t, err)

	err = migrator.Up(ctx, migrate.Latest)
	require.NoError(t, err)

	err = migrator.Down(ctx, migrate.Oldest)
	require.NoError(t, err)
}
