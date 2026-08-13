package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStockInvariant_IsEnforcedByDB — INV-1 / CLAUDE.md §3.3-3.4: доказывает,
// что защита от оверселла лежит в схеме (CHECK-констрейнт
// catalog_items_stock_invariant), а не только в удачном порядке вызовов
// Go-кода. Пишет granted_count = total_stock + 1 напрямую через пул, в обход
// всей бизнес-логики, и ожидает, что Postgres сам откажет.
func TestStockInvariant_IsEnforcedByDB(t *testing.T) {
	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 1, 90)

	_, err := pool.Exec(context.Background(),
		`UPDATE catalog_items SET granted_count = total_stock + 1 WHERE id = $1`, itemID)

	require.Error(t, err, "writing granted_count beyond total_stock directly must be rejected by the schema")
	require.Contains(t, err.Error(), "catalog_items_stock_invariant",
		"the rejection must come from the stock invariant constraint specifically, not some other error")
}
