package versioner

import (
	"context"
	"database/sql"
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

	Postgres[T Transaction] struct {
		DB Database[T]
	}

	Versioner[T Transaction] struct {
		Transaction T
	}
)

func From[T Transaction](db Database[T]) *Postgres[T] {
	return &Postgres[T]{DB: db}
}

func (p *Postgres[T]) Transaction(ctx context.Context, handler func(tx *Versioner[T]) error) error {
	tx, err := p.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := handler(&Versioner[T]{tx}); err != nil {
		return err
	}

	return tx.Commit()
}

func (p *Versioner[T]) GetCurrentVersion(ctx context.Context) (int, error) {
	var version int

	row, err := p.Transaction.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return 0, err
	}
	defer row.Close()

	if row.Next() {
		if err := row.Scan(&version); err != nil {
			return 0, err
		}
	} else {
		return 0, sql.ErrNoRows
	}

	return version, nil
}

func (p *Versioner[T]) SetVersion(ctx context.Context, version int) error {
	_, err := p.Transaction.Exec("UPDATE schema_migrations SET version = $1", version)
	return err
}
