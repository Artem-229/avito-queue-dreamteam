package repository

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"avito-queue/internal/domain"
)

// TestExpireAllOverdue_ConcurrentWithItemScopedExpiry_NoDoubleDecrement —
// прямой ответ на возражение Артёма: если ExpireOverdue(itemID) (обычный
// путь Reconcile) и глобальный ExpireAllOverdue запускаются одновременно на
// ОДНОМ товаре, granted_count не должен упасть дважды за одну и ту же
// просроченную запись. Гарантия — не явная блокировка, а перепроверка
// WHERE status='granted' после ожидания на заблокированной Postgres строке:
// кто бы ни выполнился вторым, уже обработанная запись просто не пройдёт
// условие повторно.
func TestExpireAllOverdue_ConcurrentWithItemScopedExpiry_NoDoubleDecrement(t *testing.T) {
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
		`INSERT INTO catalog_items (id, name, price_kopecks, total_stock, granted_count, category, seller_name, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, now())`,
		itemID, "expiry race test item", 10000, 10, 5, "test-category", "test-seller")
	require.NoError(t, err, "insert test catalog item with granted_count=5")
	defer pool.Exec(context.Background(), `DELETE FROM catalog_items WHERE id = $1`, itemID)

	// Пять granted-записей, все уже просрочены — ровно то количество,
	// на которое выставлен granted_count товара.
	for i := 0; i < 5; i++ {
		_, err = pool.Exec(ctx,
			`INSERT INTO queue_entries (id, item_id, user_id, status, granted_at, expires_at)
			 VALUES ($1, $2, $3, $4, now() - interval '10 minutes', now() - interval '5 minutes')`,
			uuid.New(), itemID, uuid.New(), domain.QueueEntryStatusGranted)
		require.NoError(t, err, "insert overdue granted entry")
	}
	defer pool.Exec(context.Background(), `DELETE FROM queue_entries WHERE item_id = $1`, itemID)

	repo := NewQueueEntryRepo(pool)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		// Путь Reconcile: ExpireOverdue требует транзакции с LockItem.
		err := repo.InTx(ctx, func(ctx context.Context) error {
			if _, err := lockItemForTest(ctx, pool, itemID); err != nil {
				return err
			}
			_, err := repo.ExpireOverdue(ctx, itemID)
			return err
		})
		if err != nil {
			t.Errorf("ExpireOverdue error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := repo.ExpireAllOverdue(ctx); err != nil {
			t.Errorf("ExpireAllOverdue error: %v", err)
		}
	}()

	wg.Wait()

	var grantedCount int
	err = pool.QueryRow(ctx, `SELECT granted_count FROM catalog_items WHERE id = $1`, itemID).Scan(&grantedCount)
	require.NoError(t, err)

	// Было 5, все 5 записей просрочены и должны быть погашены РОВНО один
	// раз каждая — итог 0, а не -5 (что было бы при двойном декременте).
	require.Equal(t, 0, grantedCount,
		"granted_count must be exactly 0 after both expiry paths ran concurrently — a negative value would mean double-decrement")

	var expiredCount int
	err = pool.QueryRow(ctx,
		`SELECT count(*) FROM queue_entries WHERE item_id = $1 AND status = $2`,
		itemID, domain.QueueEntryStatusExpired).Scan(&expiredCount)
	require.NoError(t, err)
	require.Equal(t, 5, expiredCount, "all 5 overdue entries must end up expired exactly once")
}

// lockItemForTest — минимальная замена CatalogRepo.LockItem, чтобы тест не
// тянул зависимость на весь CatalogRepository ради одной блокировки строки.
func lockItemForTest(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := db(ctx, pool).QueryRow(ctx,
		`SELECT id FROM catalog_items WHERE id = $1 FOR NO KEY UPDATE`, itemID).Scan(&id)
	return id, err
}
