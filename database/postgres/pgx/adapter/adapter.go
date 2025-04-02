package adapter

import (
	"context"
	"database/sql"
	"errors"
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
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (version BIGINT PRIMARY KEY)", p.config.tableName)

	if _, err = tx.Exec(ctx, query); err != nil {
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
	query := fmt.Sprintf("SELECT version FROM %s", p.config.tableName)

	row, err := p.Query(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to query %s: %w", p.config.tableName, err)
	}
	defer row.Close()

	if !row.Next() {
		return 0, row.Err()
	}

	var version int64

	if err := row.Scan(&version); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("failed to scan version: %w", err)
	}
	return version, nil
}

func (p *Versioner) SetVersion(ctx context.Context, version int64) error {
	query := fmt.Sprintf("UPDATE %s SET version = $1", p.config.tableName)

	if cmd, err := p.Exec(ctx, query, version); err != nil {
		return fmt.Errorf("failed to update version: %w", err)
	} else if cmd.RowsAffected() == 0 {
		query = fmt.Sprintf("INSERT INTO %s VALUES ($1)", p.config.tableName)
		_, err := p.Exec(ctx, query, version)
		if err != nil {
			return fmt.Errorf("failed to insert default version: %w", err)
		}
	}
	return nil
}
