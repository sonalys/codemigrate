package adapter

import (
	"context"
	"database/sql"
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
)

func From[T Transaction](db Database[T]) *Postgres[T] {
	return &Postgres[T]{db: db}
}

func (p *Postgres[T]) Transaction(ctx context.Context, handler func(tx *Versioner[T]) error) error {
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("CREATE TABLE IF NOT EXISTS $1 (version BIGINT PRIMARY KEY)", p.config.tableName)
	if err != nil {
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
	var version int64

	row, err := p.Tx.Query("SELECT version FROM $1", p.config.tableName)
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

func (p *Versioner[T]) SetVersion(ctx context.Context, version int64) error {
	_, err := p.Tx.Exec("UPDATE $1 SET version = $1", p.config.tableName, version)
	if err != nil {
		return fmt.Errorf("failed to update version: %w", err)
	}
	return nil
}
