package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func extractTx(ctx context.Context) (pgx.Tx, bool) {
	if tx, ok := ctx.Value(key).(pgx.Tx); ok {
		return tx, true
	}
	return nil, false
}
