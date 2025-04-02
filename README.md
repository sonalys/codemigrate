# CodeMigrate

[![CI](https://github.com/sonalys/codemigrate/actions/workflows/ci.yml/badge.svg)](https://github.com/sonalys/codemigrate/actions/workflows/ci.yml)

CodeMigrate is a Go library designed to simplify database schema migrations. It provides a flexible and extensible framework for managing migrations across different database backends, including PostgreSQL with both `pq` and `pgx` drivers.

## Features

- **Database-agnostic**: Supports multiple database drivers (`pq` and `pgx`).
- **Versioning**: Tracks and manages schema versions.
- **Custom Migrations**: Easily define `Up` and `Down` migrations.
- **Transaction Support**: Ensures migrations are applied atomically.
- **Error Handling**: Provides meaningful errors for migration conflicts, missing migrations, and more.

## Project Structure

The project is organized as follows:

- `migrate/`: Core migration logic and abstractions.
- `database/postgres/pq/`: PostgreSQL adapter using the `pq` driver.
- `database/postgres/pgx/`: PostgreSQL adapter using the `pgx` driver.
- `examples/`: Example usage for both `pq` and `pgx` adapters.

## Installation

To use CodeMigrate in your project, add the required module to your `go.mod` file:

```bash
go get github.com/sonalys/codemigrate/migrate
```

For PostgreSQL support, include the desired adapter:

```bash
go get github.com/sonalys/codemigrate/database/postgres/pq
# or
go get github.com/sonalys/codemigrate/database/postgres/pgx
```

## Usage

### Define a Migration

Create a struct that implements the `Migration` interface:

```go
type migration_0001 struct{}

func (m *migration_0001) Version() int64 {
	return 1
}

func (m *migration_0001) Up(ctx context.Context, tx *adapter.Versioner[*sql.Tx]) error {
	_, err := tx.Tx.Exec("CREATE TABLE IF NOT EXISTS test (id SERIAL PRIMARY KEY, name TEXT)")
	return err
}

func (m *migration_0001) Down(ctx context.Context, tx *adapter.Versioner[*sql.Tx]) error {
	_, err := tx.Tx.Exec("DROP TABLE IF EXISTS test")
	return err
}
```

### Initialize the Migrator

Use the appropriate adapter to initialize the migrator:

```go
import (
	"github.com/sonalys/codemigrate/database/postgres/pq/adapter"
	"github.com/sonalys/codemigrate/migrate"
)

conn, err := sql.Open("postgres", "your_connection_string")
if err != nil {
	log.Fatal(err)
}

db := adapter.From(conn, adapter.WithTableName("custom_version_table"))

migrator, err := migrate.New(db, &migration_0001{})
if err != nil {
	log.Fatal(err)
}
```

### Apply Migrations

Run migrations using the `Up` or `Down` methods:

```go
err = migrator.Up(context.Background(), migrate.Latest)
if err != nil {
	log.Fatal(err)
}

err = migrator.Down(context.Background(), migrate.Oldest)
if err != nil {
	log.Fatal(err)
}
```

### Use Migrations from Files

You can also define migrations using SQL scripts stored in files. Here's an example:

```go
import (
	"context"
	"embed"
	"log"

	"github.com/sonalys/codemigrate/database/postgres/pgx/adapter"
	"github.com/sonalys/codemigrate/migrate"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func main() {
	conn, err := pgx.Connect(context.Background(), "your_connection_string")
	if err != nil {
		log.Fatal(err)
	}

	db := adapter.From(conn, adapter.WithTableName("custom_version_table"))

	migration, err := adapter.NewScriptMigrationFromFile(
		1,
		migrationFiles,
		"migrations/0001_up.sql",
		"migrations/0001_down.sql",
	)
	if err != nil {
		log.Fatal(err)
	}

	migrator, err := migrate.New(db, migration)
	if err != nil {
		log.Fatal(err)
	}

	err = migrator.Up(context.Background(), migrate.Latest)
	if err != nil {
		log.Fatal(err)
	}
}
```

In this example, the `migrations/001_up.sql` and `migrations/001_down.sql` files contain the SQL scripts for applying and reverting the migration, respectively.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
