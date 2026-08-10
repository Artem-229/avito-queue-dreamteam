package services

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"avito-queue/internal/domain"
	"avito-queue/internal/infra/postgres/repository"
)

// testDBPool открывает пул к тестовой БД (TEST_DATABASE_DSN, дефолт как в
// internal/infra/postgres/repository/purchase_right_repo_test.go). В отличие
// от того файла, здесь пул сначала пингуется с коротким таймаутом: если
// Postgres не поднят (например нет docker compose в песочнице), тест
// аккуратно скипается через t.Skip, а не висит/падает по общему таймауту
// пакета. Не продуктовый код — вспомогательная функция только для тестов.
func testDBPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/avito_queue?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err, "parsing test db dsn")

	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		t.Skipf("test database is not reachable at %s (start it via docker compose): %v", dsn, err)
	}

	t.Cleanup(pool.Close)
	return pool
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newTestQueueService собирает QueueService на настоящих репозиториях
// (не моках): интеграционные тесты в этом пакете проверяют реальный SQL,
// реальные блокировки строк и реальный констрейнт, а не поведение сервиса
// в изоляции от БД.
func newTestQueueService(pool *pgxpool.Pool) *QueueService {
	return NewQueueService(
		repository.NewQueueRepo(pool),
		repository.NewPurchaseRightRepo(pool),
		repository.NewCatalogRepository(pool),
		discardLogger(),
	)
}

// newTestPurchaseRight собирает PurchaseRight-сервис на настоящих
// репозиториях, используя queueSvc как queueAdvancer (EnsureAdvanced) —
// тот же контракт T-09/D-03, что использует продовый DI в internal/app.
func newTestPurchaseRight(pool *pgxpool.Pool, queueSvc *QueueService) *PurchaseRight {
	return NewPurchaseRight(
		repository.NewPurchaseRightRepo(pool),
		repository.NewQueueRepo(pool),
		repository.NewCatalogRepository(pool),
		queueSvc,
	)
}

// insertCatalogItem создаёт тестовый товар со своим uuid и регистрирует
// очистку по завершении теста через t.Cleanup — каждый тест работает на
// своих данных, без общего изменяемого состояния между тестами.
func insertCatalogItem(t *testing.T, pool *pgxpool.Pool, totalStock, holdTTLSeconds int) uuid.UUID {
	t.Helper()

	ctx := context.Background()
	id := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO catalog_items (id, name, price_kopecks, total_stock, hold_ttl_seconds, category, seller_name, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())`,
		id, "test-writer item "+id.String(), 10000, totalStock, holdTTLSeconds, "test-category", "test-seller")
	require.NoError(t, err, "insert test catalog item")

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM purchase_rights WHERE item_id = $1`, id)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM queue WHERE item_id = $1`, id)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM catalog_items WHERE id = $1`, id)
	})

	return id
}

func countQueueByStatus(t *testing.T, pool *pgxpool.Pool, itemID uuid.UUID, status domain.QueueStatus) int {
	t.Helper()
	var count int
	err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM queue WHERE item_id = $1 AND status = $2`, itemID, status).Scan(&count)
	require.NoError(t, err)
	return count
}

func countPurchaseRightsByStatus(t *testing.T, pool *pgxpool.Pool, itemID uuid.UUID, status domain.PurchaseRightStatus) int {
	t.Helper()
	var count int
	err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM purchase_rights WHERE item_id = $1 AND status = $2`, itemID, status).Scan(&count)
	require.NoError(t, err)
	return count
}

// mustGetQueueStatus повторяет запрос QueueRepository.GetRecord (последняя
// по created_at запись пользователя по товару), чтобы ассерты в тестах
// смотрели на то же состояние, что видит сервис.
func mustGetQueueStatus(t *testing.T, pool *pgxpool.Pool, userID, itemID uuid.UUID) domain.QueueStatus {
	t.Helper()
	var status domain.QueueStatus
	err := pool.QueryRow(context.Background(),
		`SELECT status FROM queue WHERE user_id = $1 AND item_id = $2 ORDER BY created_at DESC LIMIT 1`,
		userID, itemID).Scan(&status)
	require.NoError(t, err)
	return status
}

func catalogCounts(t *testing.T, pool *pgxpool.Pool, itemID uuid.UUID) (granted, used int) {
	t.Helper()
	err := pool.QueryRow(context.Background(),
		`SELECT granted_count, used_count FROM catalog_items WHERE id = $1`, itemID).Scan(&granted, &used)
	require.NoError(t, err)
	return granted, used
}
