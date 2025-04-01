package versioner

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
)

type (
	PGXDriver interface {
		Begin(ctx context.Context) (pgx.Tx, error)
	}

	Postgres struct {
		DB PGXDriver
	}

	Versioner struct {
		Transaction pgx.Tx
	}
)

func From(db PGXDriver) *Postgres {
	return &Postgres{DB: db}
}

func (p *Postgres) Transaction(ctx context.Context, handler func(tx *Versioner) error) error {
	tx, err := p.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := handler(&Versioner{tx}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (p *Versioner) GetCurrentVersion(ctx context.Context) (int, error) {
	var version int

	row, err := p.Transaction.Query(ctx, "SELECT version FROM schema_migrations")
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

func (p *Versioner) SetVersion(ctx context.Context, version int) error {
	_, err := p.Transaction.Exec(ctx, "UPDATE schema_migrations SET version = $1", version)
	return err
}
