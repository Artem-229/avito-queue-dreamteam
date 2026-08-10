package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB — общий контракт pgxpool.Pool и pgx.Tx.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// db возвращает транзакцию из контекста, если она есть, иначе пул.
// Единственный разрешённый способ получить исполнителя запроса в пакете repository (INV-4).
func db(ctx context.Context, pool *pgxpool.Pool) DB {
	if tx, ok := extractTx(ctx); ok {
		return tx
	}
	return pool
}
