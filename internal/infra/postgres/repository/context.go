package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type transactionKey string

// key — ключ транзакции в контексте. Репозитории достают исполнителя запроса
// только через db(ctx, pool), который смотрит сюда: прямое обращение к пулу
// внутри транзакции выполнило бы запрос вне неё (INV-4).
const key transactionKey = "transaction_key"

func extractTx(ctx context.Context) (pgx.Tx, bool) {
	if tx, ok := ctx.Value(key).(pgx.Tx); ok {
		return tx, true
	}
	return nil, false
}
