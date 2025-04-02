package adapter

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type (
	Database interface {
		Begin(ctx context.Context) (pgx.Tx, error)
	}

	Config struct {
		tableName string
	}

	Postgres struct {
		db     Database
		config Config
	}

	Versioner struct {
		pgx.Tx
		config Config
	}

	Option func(*Config)
)

// WithTableName sets the table name for the schema migrations table.
func WithTableName(name string) Option {
	return func(p *Config) {
		p.tableName = name
	}
}

func From(db Database, opts ...Option) *Postgres {
	posgtres := &Postgres{
		db: db,
		config: Config{
			tableName: "schema_migrations",
		},
	}

	for _, opt := range opts {
		opt(&posgtres.config)
	}

	return posgtres
}

func (p *Postgres) Transaction(ctx context.Context, handler func(tx *Versioner) error) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "CREATE TABLE IF NOT EXISTS $1 (version BIGINT PRIMARY KEY)", p.config.tableName)
	if err != nil {
		return fmt.Errorf("failed to create %s table: %w", p.config.tableName, err)
	}

	versioner := &Versioner{
		Tx:     tx,
		config: p.config,
	}

	if err := handler(versioner); err != nil {
		return fmt.Errorf("handler error: %w", err)
	}

	return tx.Commit(ctx)
}

func (p *Versioner) GetCurrentVersion(ctx context.Context) (int64, error) {
	var version int64

	row, err := p.Query(ctx, "SELECT version FROM $1", p.config.tableName)
	if err != nil {
		return 0, fmt.Errorf("failed to query %s: %w", p.config.tableName, err)
	}
	defer row.Close()

	if row.Next() {
		if err := row.Scan(&version); err != nil {
			return 0, fmt.Errorf("failed to scan version: %w", err)
		}
	} else {
		return 0, sql.ErrNoRows
	}

	return version, nil
}

func (p *Versioner) SetVersion(ctx context.Context, version int64) error {
	_, err := p.Exec(ctx, "UPDATE $1 SET version = $1", p.config.tableName, version)
	if err != nil {
		return fmt.Errorf("failed to update version: %w", err)
	}
	return nil
}
