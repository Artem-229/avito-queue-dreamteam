package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"avito-queue/internal/domain"
)

// Общие хелперы интеграционных тестов репозиториев. DSN берётся из
// TEST_DATABASE_DSN; без поднятой базы тесты скипаются, а не падают.

// testDSNFallback — адрес базы из docker compose, единственного описанного в
// README способа её поднять: порт 5433 на 127.0.0.1. Стандартный 5432 в этой
// роли хуже дважды — на нём тесты молча скипались бы при штатно поднятом
// compose, а на машине с локально установленным Postgres тестовые товары
// поехали бы в чужую базу. CI задаёт TEST_DATABASE_DSN явно, там свой порт.
const testDSNFallback = "postgres://postgres:postgres@localhost:5433/avito_queue?sslmode=disable"

func testDBPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = testDSNFallback
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err, "parsing test db dsn")

	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		t.Skipf("тестовая база недоступна по %s (поднимите её через docker compose): %v", dsn, err)
	}

	t.Cleanup(pool.Close)

	return pool
}

func insertTestItem(t *testing.T, pool *pgxpool.Pool, totalStock, holdTTLSeconds int) uuid.UUID {
	t.Helper()

	ctx := context.Background()
	itemID := uuid.New()

	_, err := pool.Exec(ctx, `
		INSERT INTO catalog_items (id, name, price_kopecks, total_stock, hold_ttl_seconds, category, seller_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		itemID, "repo test item "+itemID.String(), 10000, totalStock, holdTTLSeconds, "test-category", "test-seller")
	require.NoError(t, err, "insert test catalog item")

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM queue_entries WHERE item_id = $1`, itemID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM catalog_items WHERE id = $1`, itemID)
	})

	return itemID
}

// enterQueue кладёт пользователя в очередь напрямую через репозиторий и
// возвращает его id. Отдельный хелпер нужен, потому что почти каждому тесту
// репозитория требуется заранее подготовленное участие.
func enterQueue(t *testing.T, repo *QueueEntryRepository, itemID uuid.UUID) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	require.NoError(t, repo.Enter(context.Background(), userID, itemID))

	return userID
}

// grantAll выдаёт права первым slots ожидающим. GrantNext требует транзакции
// с уже взятой блокировкой товара — здесь она берётся так же, как это делает
// Reconcile (INV-3).
func grantAll(t *testing.T, repo *QueueEntryRepository, catalog *CatalogRepository, itemID uuid.UUID, slots int) []uuid.UUID {
	t.Helper()

	var granted []uuid.UUID
	err := repo.InTx(context.Background(), func(ctx context.Context) error {
		if _, err := catalog.LockItem(ctx, itemID); err != nil {
			return err
		}

		users, err := repo.GrantNext(ctx, itemID, slots)
		granted = users

		return err
	})
	require.NoError(t, err)

	return granted
}

func statusOf(t *testing.T, pool *pgxpool.Pool, userID, itemID uuid.UUID) domain.QueueEntryStatus {
	t.Helper()

	var status domain.QueueEntryStatus
	err := pool.QueryRow(context.Background(), `
		SELECT status FROM queue_entries
		 WHERE user_id = $1 AND item_id = $2
		 ORDER BY created_at DESC
		 LIMIT 1`,
		userID, itemID).Scan(&status)
	require.NoError(t, err)

	return status
}

func itemCounters(t *testing.T, pool *pgxpool.Pool, itemID uuid.UUID) (granted, used int) {
	t.Helper()

	err := pool.QueryRow(context.Background(),
		`SELECT granted_count, used_count FROM catalog_items WHERE id = $1`, itemID).Scan(&granted, &used)
	require.NoError(t, err)

	return granted, used
}
