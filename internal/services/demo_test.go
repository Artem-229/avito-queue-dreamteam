package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"avito-queue/internal/domain"
	"avito-queue/internal/infra/postgres/repository"
)

func newTestDemoService(pool *pgxpool.Pool, queue *QueueEntryService) *DemoService {
	return NewDemoService(repository.NewQueueEntryRepo(pool), repository.NewCatalogRepository(pool), queue)
}

// TestSimulate_EntersEveryoneWithoutOversell — симулятор заводит покупателей
// через настоящий Entry, а не подделывает состояние. Значит, на нём должны
// выполняться те же правила: в очередь попадают все, право получают только
// столько, сколько единиц в тираже.
func TestSimulate_EntersEveryoneWithoutOversell(t *testing.T) {
	t.Parallel()

	const (
		totalStock = 2
		count      = 25
	)

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, totalStock, 120)
	queue := newTestQueueService(pool)
	demo := newTestDemoService(pool, queue)

	joined, failed, err := demo.Simulate(context.Background(), itemID, count)
	require.NoError(t, err)

	t.Logf("симуляция: заявлено %d, вошли %d, отказов %d", count, joined, failed)
	require.Equal(t, count, joined, "все заявленные претенденты обязаны оказаться в очереди")
	require.Zero(t, failed)

	require.Equal(t, totalStock, countEntries(t, pool, itemID, domain.QueueEntryStatusGranted))
	require.Equal(t, count-totalStock, countEntries(t, pool, itemID, domain.QueueEntryStatusWaiting))
	requireCountersMatchEntries(t, pool, itemID, totalStock)
}

// TestSimulate_UnknownItem — симулятор по несуществующему товару не должен
// молча отчитываться об успехе.
func TestSimulate_UnknownItem(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	queue := newTestQueueService(pool)
	demo := newTestDemoService(pool, queue)

	joined, failed, err := demo.Simulate(context.Background(), uuid.New(), 3)
	require.NoError(t, err)
	require.Zero(t, joined)
	require.Equal(t, 3, failed, "вход на несуществующий товар — отказ, а не участие")
}

// TestExpireNow_BurnsRightsAndHandsSlotOver — демо-ручка для защиты: она
// обязана не просто сдвинуть сроки, а сразу отдать освободившийся слот
// следующему. Иначе на демонстрации пришлось бы ждать, пока кто-нибудь
// случайно опросит статус.
func TestExpireNow_BurnsRightsAndHandsSlotOver(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 1, 120)
	queue := newTestQueueService(pool)
	demo := newTestDemoService(pool, queue)
	ctx := context.Background()

	holder, next := uuid.New(), uuid.New()

	holderResp, err := queue.Entry(ctx, holder, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueEntryStatusGranted, holderResp.Status)

	nextResp, err := queue.Entry(ctx, next, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueEntryStatusWaiting, nextResp.Status)

	require.NoError(t, demo.ExpireNow(ctx, itemID))

	require.Equal(t, domain.QueueEntryStatusExpired, statusOf(t, pool, holder, itemID),
		"право первого обязано сгореть")
	require.Equal(t, domain.QueueEntryStatusGranted, statusOf(t, pool, next, itemID),
		"слот обязан уйти следующему в тот же вызов, без ожидания опроса статуса")
	requireCountersMatchEntries(t, pool, itemID, 1)
}

// TestExpireNow_UnknownItem — блокировка несуществующего товара обязана дать
// доменную ошибку, а не пройти как успех.
func TestExpireNow_UnknownItem(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	queue := newTestQueueService(pool)
	demo := newTestDemoService(pool, queue)

	require.ErrorIs(t, demo.ExpireNow(context.Background(), uuid.New()), domain.ErrNoItemFound)
}
