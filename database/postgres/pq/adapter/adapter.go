package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type (
	Transaction interface {
		Rollback() error
		Commit() error

		Query(query string, args ...interface{}) (*sql.Rows, error)
		Exec(query string, args ...interface{}) (sql.Result, error)
	}

	Database[T Transaction] interface {
		Begin() (T, error)
	}

	Config struct {
		tableName string
	}

	Postgres[T Transaction] struct {
		db     Database[T]
		config Config
	}

	Versioner[T Transaction] struct {
		Tx     T
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

func From[T Transaction](db Database[T], opts ...Option) *Postgres[T] {
	postgres := &Postgres[T]{
		db: db,
		config: Config{
			tableName: "schema_migrations",
		},
	}

	for _, opt := range opts {
		opt(&postgres.config)
	}

	return postgres
}

func (p *Postgres[T]) Transaction(ctx context.Context, handler func(tx *Versioner[T]) error) error {
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (version BIGINT PRIMARY KEY)", p.config.tableName)

	if _, err = tx.Exec(query); err != nil {
		return fmt.Errorf("failed to create %s table: %w", p.config.tableName, err)
	}

	versioner := &Versioner[T]{
		Tx:     tx,
		config: p.config,
	}

	if err := handler(versioner); err != nil {
		return fmt.Errorf("handler error: %w", err)
	}

	return tx.Commit()
}

func (p *Versioner[T]) GetCurrentVersion(ctx context.Context) (int64, error) {
	query := fmt.Sprintf("SELECT version FROM %s", p.config.tableName)

	row, err := p.Tx.Query(query)
	if err != nil {
		return 0, fmt.Errorf("failed to query %s: %w", p.config.tableName, err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return 0, row.Err()
	}

	var version int64

	if err := row.Scan(&version); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("failed to scan version: %w", err)
	}
	return version, nil
}

func (p *Versioner[T]) SetVersion(ctx context.Context, version int64) error {
	query := fmt.Sprintf("UPDATE %s SET version = $1", p.config.tableName)

	cmd, err := p.Tx.Exec(query, version)
	if err != nil {
		return fmt.Errorf("failed to update version: %w", err)
	}

	rowsAffected, err := cmd.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get exec info: %w", err)
	}

	if rowsAffected == 0 {
		query = fmt.Sprintf("INSERT INTO %s VALUES ($1)", p.config.tableName)
		_, err := p.Tx.Exec(query, version)
		if err != nil {
			return fmt.Errorf("failed to insert default version: %w", err)
		}
	}
	return nil
}
