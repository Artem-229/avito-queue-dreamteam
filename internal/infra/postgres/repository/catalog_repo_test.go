package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// TestStockInvariant_IsEnforcedByDB — INV-1 / CLAUDE.md §3.3-3.4: доказывает,
// что защита от оверселла лежит в схеме (CHECK-констрейнт
// catalog_items_stock_invariant), а не только в удачном порядке вызовов
// Go-кода. Пишет granted_count = total_stock + 1 напрямую через пул, в обход
// всей бизнес-логики, и ожидает, что Postgres сам откажет.
func TestStockInvariant_IsEnforcedByDB(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/avito_queue?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err, "parsing test db dsn")
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		t.Skipf("test database is not reachable at %s (start it via docker compose): %v", dsn, err)
	}

	ctx := context.Background()
	itemID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO catalog_items (id, name, price_kopecks, total_stock, category, seller_name, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, now())`,
		itemID, "stock invariant test item", 10000, 1, "test-category", "test-seller")
	require.NoError(t, err, "insert test catalog item")
	defer pool.Exec(context.Background(), `DELETE FROM catalog_items WHERE id = $1`, itemID)

	_, err = pool.Exec(ctx,
		`UPDATE catalog_items SET granted_count = total_stock + 1 WHERE id = $1`, itemID)

	require.Error(t, err, "writing granted_count beyond total_stock directly must be rejected by the schema")
	require.Contains(t, err.Error(), "catalog_items_stock_invariant",
		"the rejection must come from the stock invariant constraint specifically, not some other error")
}
