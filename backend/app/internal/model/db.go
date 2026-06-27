package model

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/lib/pq"
)

func OpenPostgres(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, errors.New("PostgreSQL DSN 未配置")
	}
	return sql.Open("postgres", dsn)
}

func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
