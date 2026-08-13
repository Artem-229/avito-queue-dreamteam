package services

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"avito-queue/internal/domain"
	"avito-queue/internal/infra/postgres/repository"
)

// Тесты этого пакета интеграционные: сервис очереди проверяется поверх
// настоящего Postgres, а не поверх моков репозитория. Мок здесь ничего не
// доказал бы — вся защита от оверселла держится на блокировке строки товара,
// CTE-обновлении счётчиков и CHECK-констрейнте, то есть ровно на том, что мок
// заменяет собой заглушкой.
//
// DSN берётся из TEST_DATABASE_DSN. Без поднятой базы тесты скипаются, а не
// падают: см. README «Тесты».

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

// newTestQueueService собирает QueueEntryService так же, как его собирает
// internal/app: настоящие репозитории и настоящий калькулятор шанса. Подмена
// зависимостей здесь означала бы, что тест проверяет не тот объект, который
// поедет в прод.
func newTestQueueService(pool *pgxpool.Pool) *QueueEntryService {
	queueRepo := repository.NewQueueEntryRepo(pool)

	return NewQueueEntryService(queueRepo, repository.NewCatalogRepository(pool), NewChanceCalculator(queueRepo))
}

// insertTestItem заводит товар со своим uuid и убирает его за собой. Каждый
// тест работает на собственных данных: общего изменяемого состояния между
// тестами нет, поэтому их можно гонять параллельно и в любом порядке.
func insertTestItem(t *testing.T, pool *pgxpool.Pool, totalStock, holdTTLSeconds int) uuid.UUID {
	t.Helper()

	ctx := context.Background()
	itemID := uuid.New()

	_, err := pool.Exec(ctx, `
		INSERT INTO catalog_items (id, name, price_kopecks, total_stock, hold_ttl_seconds, category, seller_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		itemID, "test item "+itemID.String(), 10000, totalStock, holdTTLSeconds, "test-category", "test-seller")
	require.NoError(t, err, "insert test catalog item")

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM queue_entries WHERE item_id = $1`, itemID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM catalog_items WHERE id = $1`, itemID)
	})

	return itemID
}

func countEntries(t *testing.T, pool *pgxpool.Pool, itemID uuid.UUID, status domain.QueueEntryStatus) int {
	t.Helper()

	var count int
	err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM queue_entries WHERE item_id = $1 AND status = $2`, itemID, status).Scan(&count)
	require.NoError(t, err)

	return count
}

// itemCounters — счётчики товара. Их сверка с реальным содержимым
// queue_entries и есть проверка INV-5: счётчик двигается тем же оператором,
// что и статус, поэтому разойтись они не имеют права никогда.
func itemCounters(t *testing.T, pool *pgxpool.Pool, itemID uuid.UUID) (granted, used int) {
	t.Helper()

	err := pool.QueryRow(context.Background(),
		`SELECT granted_count, used_count FROM catalog_items WHERE id = $1`, itemID).Scan(&granted, &used)
	require.NoError(t, err)

	return granted, used
}

// requireCountersMatchEntries — INV-1 и INV-5 одной проверкой: счётчики товара
// совпадают с фактическим числом строк в соответствующих статусах, и их сумма
// не превышает тираж.
func requireCountersMatchEntries(t *testing.T, pool *pgxpool.Pool, itemID uuid.UUID, totalStock int) {
	t.Helper()

	granted, used := itemCounters(t, pool, itemID)
	require.Equal(t, countEntries(t, pool, itemID, domain.QueueEntryStatusGranted), granted,
		"granted_count разошёлся с числом записей в статусе granted (INV-5)")
	require.Equal(t, countEntries(t, pool, itemID, domain.QueueEntryStatusPurchased), used,
		"used_count разошёлся с числом записей в статусе purchased (INV-5)")
	require.LessOrEqual(t, granted+used, totalStock,
		"granted + used превысил тираж — это и есть оверселл (INV-1)")
}

// expireGrants двигает срок всех выданных прав на товар в прошлое. Истечение
// через time.Sleep сделало бы тест медленным и хрупким, а проверяется здесь не
// точность часов, а реакция на просроченное право.
func expireGrants(t *testing.T, pool *pgxpool.Pool, itemID uuid.UUID) {
	t.Helper()

	_, err := pool.Exec(context.Background(), `
		UPDATE queue_entries
		   SET expires_at = now() - interval '1 second'
		 WHERE item_id = $1 AND status = $2`,
		itemID, domain.QueueEntryStatusGranted)
	require.NoError(t, err, "forcing grant expiry directly in the DB")
}

// statusOf — последнее участие пользователя по товару, тем же запросом, каким
// его видит сервис (GetEntry сортирует по created_at DESC).
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
