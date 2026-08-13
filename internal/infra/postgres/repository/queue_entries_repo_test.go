package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"avito-queue/internal/domain"
)

// Тесты репозитория очереди проверяют вход и выход каждого метода на живой
// базе: здесь важен именно SQL — частичный уникальный индекс, порядок FIFO,
// сравнение сроков силами Postgres и CTE, двигающий счётчики товара.

// TestEnter_SecondActiveEntryIsRejected — частичный уникальный индекс
// queue_entries_active должен пропускать ровно одно активное участие на пару
// пользователь-товар. Предикат индекса обязан дословно совпадать с ON CONFLICT
// в Enter, иначе Postgres не выведет индекс и вставка упадёт ошибкой вместо
// тихого DO NOTHING.
func TestEnter_SecondActiveEntryIsRejected(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	itemID := insertTestItem(t, pool, 5, 120)
	ctx := context.Background()

	userID := enterQueue(t, repo, itemID)

	err := repo.Enter(ctx, userID, itemID)
	require.ErrorIs(t, err, domain.ErrUserAlreadyInQueue,
		"повторный вход при активном участии — не ошибка вставки, а сигнал «вы уже в очереди»")

	waiting, err := repo.CountByStatus(ctx, itemID, domain.QueueEntryStatusWaiting)
	require.NoError(t, err)
	require.Equal(t, 1, waiting, "вторая строка участия появиться не должна")
}

// TestEnter_FillsInitialPosition — позиция на входе считается тем же запросом,
// что и вставка: постфактум она невычислима, а на ней держится полоса
// продвижения очереди.
func TestEnter_FillsInitialPosition(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	itemID := insertTestItem(t, pool, 5, 120)
	ctx := context.Background()

	for expected := 1; expected <= 3; expected++ {
		userID := enterQueue(t, repo, itemID)

		entry, err := repo.GetEntry(ctx, userID, itemID)
		require.NoError(t, err)
		require.NotNil(t, entry.InitialPosition, "позиция на входе обязана заполняться при вставке")
		require.Equal(t, expected, *entry.InitialPosition,
			"каждый следующий вошедший встаёт в хвост очереди")
	}
}

// TestGrantNext_FollowsFIFOAndMovesCounters — выдача идёт по (created_at,
// user_id): тем же ключом считается позиция в Snapshot, иначе показанное место
// разошлось бы с реальным порядком выдачи (INV-10).
func TestGrantNext_FollowsFIFOAndMovesCounters(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	catalog := NewCatalogRepository(pool)
	itemID := insertTestItem(t, pool, 3, 120)
	ctx := context.Background()

	order := make([]uuid.UUID, 0, 4)
	for range 4 {
		order = append(order, enterQueue(t, repo, itemID))
	}

	granted := grantAll(t, repo, catalog, itemID, 2)
	require.Equal(t, order[:2], granted, "право обязано уйти двум первым по времени входа")

	for _, userID := range order[:2] {
		require.Equal(t, domain.QueueEntryStatusGranted, statusOf(t, pool, userID, itemID))

		entry, err := repo.GetEntry(ctx, userID, itemID)
		require.NoError(t, err)
		require.NotNil(t, entry.GrantedAt, "у выданного права есть момент выдачи")
		require.NotNil(t, entry.ExpiresAt, "у выданного права есть срок (констрейнт granted_has_deadline)")
		require.True(t, entry.ExpiresAt.After(*entry.GrantedAt), "срок права истекает позже момента выдачи")
	}

	for _, userID := range order[2:] {
		require.Equal(t, domain.QueueEntryStatusWaiting, statusOf(t, pool, userID, itemID))
	}

	grantedCount, usedCount := itemCounters(t, pool, itemID)
	require.Equal(t, 2, grantedCount, "счётчик товара двигается тем же оператором, что и статус (INV-5)")
	require.Zero(t, usedCount)
}

// TestGrantNext_WithoutSlotsChangesNothing — защита от вызова с нулём слотов:
// запрос не должен ни выдавать прав, ни трогать счётчики.
func TestGrantNext_WithoutSlotsChangesNothing(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	catalog := NewCatalogRepository(pool)
	itemID := insertTestItem(t, pool, 1, 120)

	enterQueue(t, repo, itemID)
	require.Empty(t, grantAll(t, repo, catalog, itemID, 0))

	grantedCount, usedCount := itemCounters(t, pool, itemID)
	require.Zero(t, grantedCount)
	require.Zero(t, usedCount)
}

// TestExpireOverdue_GasitProsrochennyeAndReturnsUsers — сравнение с now()
// делает Postgres, та же база, что считала expires_at при выдаче: сравнивать
// часы двух машин здесь нельзя (INV-6).
func TestExpireOverdue_ReturnsUsersAndReleasesSlots(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	catalog := NewCatalogRepository(pool)
	itemID := insertTestItem(t, pool, 2, 120)
	ctx := context.Background()

	for range 2 {
		enterQueue(t, repo, itemID)
	}
	granted := grantAll(t, repo, catalog, itemID, 2)
	require.Len(t, granted, 2)

	_, err := pool.Exec(ctx, `
		UPDATE queue_entries SET expires_at = now() - interval '1 second'
		 WHERE item_id = $1 AND user_id = $2`, itemID, granted[0])
	require.NoError(t, err)

	var expired []uuid.UUID
	require.NoError(t, repo.InTx(ctx, func(txCtx context.Context) error {
		if _, err := catalog.LockItem(txCtx, itemID); err != nil {
			return err
		}

		expired, err = repo.ExpireOverdue(txCtx, itemID)

		return err
	}))

	require.Equal(t, []uuid.UUID{granted[0]}, expired,
		"гаснут только права с истёкшим сроком, живое право трогать нельзя")
	require.Equal(t, domain.QueueEntryStatusExpired, statusOf(t, pool, granted[0], itemID))
	require.Equal(t, domain.QueueEntryStatusGranted, statusOf(t, pool, granted[1], itemID))

	grantedCount, _ := itemCounters(t, pool, itemID)
	require.Equal(t, 1, grantedCount, "сгоревшее право обязано вернуть единицу в пул")
}

// TestExpireOverdue_RequiresTransaction — метод меняет распределение прав,
// поэтому обязан выполняться под блокировкой товара. Вызов вне транзакции —
// ошибка, а не «сработает и так».
func TestExpireOverdue_RequiresTransaction(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	itemID := insertTestItem(t, pool, 1, 120)

	_, err := repo.ExpireOverdue(context.Background(), itemID)
	require.ErrorIs(t, err, domain.ErrNoTx)

	_, err = repo.GrantNext(context.Background(), itemID, 1)
	require.ErrorIs(t, err, domain.ErrNoTx)
}

// TestMarkPurchased_Cases — вход и выход единственного метода, превращающего
// право в покупку. false без ошибки означает «права нет, оно уже потрачено или
// просрочено» — три разные ситуации, одинаково безопасные для тиража.
func TestMarkPurchased_Cases(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	catalog := NewCatalogRepository(pool)
	ctx := context.Background()

	t.Run("действующее право тратится", func(t *testing.T) {
		itemID := insertTestItem(t, pool, 1, 120)
		enterQueue(t, repo, itemID)
		userID := grantAll(t, repo, catalog, itemID, 1)[0]

		bought, err := repo.MarkPurchased(ctx, userID, itemID)
		require.NoError(t, err)
		require.True(t, bought)

		grantedCount, usedCount := itemCounters(t, pool, itemID)
		require.Zero(t, grantedCount, "потраченное право перестаёт удерживать единицу")
		require.Equal(t, 1, usedCount)
	})

	t.Run("повторная трата не проходит", func(t *testing.T) {
		itemID := insertTestItem(t, pool, 1, 120)
		enterQueue(t, repo, itemID)
		userID := grantAll(t, repo, catalog, itemID, 1)[0]

		_, err := repo.MarkPurchased(ctx, userID, itemID)
		require.NoError(t, err)

		bought, err := repo.MarkPurchased(ctx, userID, itemID)
		require.NoError(t, err)
		require.False(t, bought, "второй раз то же право потратить нельзя")

		_, usedCount := itemCounters(t, pool, itemID)
		require.Equal(t, 1, usedCount, "счётчик покупок не должен вырасти дважды")
	})

	t.Run("просроченное право не покупка", func(t *testing.T) {
		itemID := insertTestItem(t, pool, 1, 120)
		enterQueue(t, repo, itemID)
		userID := grantAll(t, repo, catalog, itemID, 1)[0]

		_, err := pool.Exec(ctx, `
			UPDATE queue_entries SET expires_at = now() - interval '1 second'
			 WHERE item_id = $1 AND user_id = $2`, itemID, userID)
		require.NoError(t, err)

		bought, err := repo.MarkPurchased(ctx, userID, itemID)
		require.NoError(t, err)
		require.False(t, bought, "право, истёкшее миллисекундой раньше, покупкой не становится")
	})

	t.Run("ожидающий права не имеет", func(t *testing.T) {
		itemID := insertTestItem(t, pool, 1, 120)
		userID := enterQueue(t, repo, itemID)

		bought, err := repo.MarkPurchased(ctx, userID, itemID)
		require.NoError(t, err)
		require.False(t, bought)
	})
}

// TestUpdateStatus_IsCAS — переход применяется только из ожидаемого состояния
// (INV-11). Ноль затронутых строк — конфликт, а не «нечего делать»: иначе
// гонка двух переходов прошла бы незамеченной.
func TestUpdateStatus_IsCAS(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	itemID := insertTestItem(t, pool, 1, 120)
	ctx := context.Background()

	userID := enterQueue(t, repo, itemID)

	require.NoError(t, repo.UpdateStatus(ctx, userID, itemID,
		domain.QueueEntryStatusWaiting, domain.QueueEntryStatusCancelled))
	require.Equal(t, domain.QueueEntryStatusCancelled, statusOf(t, pool, userID, itemID))

	err := repo.UpdateStatus(ctx, userID, itemID,
		domain.QueueEntryStatusWaiting, domain.QueueEntryStatusCancelled)
	require.ErrorIs(t, err, domain.ErrStatusConflict,
		"повторный переход из уже пройденного состояния обязан быть конфликтом")

	entry, err := repo.GetEntry(ctx, userID, itemID)
	require.NoError(t, err)
	require.NotNil(t, entry.ResolvedAt, "переход в терминальный статус проставляет момент завершения")
}

// TestUpdateStatus_MovesCountersOnGrantedTransitions — переход держателя права
// в отменённое участие обязан вернуть единицу тиража тем же оператором.
func TestUpdateStatus_MovesCountersOnGrantedTransitions(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	catalog := NewCatalogRepository(pool)
	itemID := insertTestItem(t, pool, 1, 120)

	enterQueue(t, repo, itemID)
	userID := grantAll(t, repo, catalog, itemID, 1)[0]

	grantedCount, _ := itemCounters(t, pool, itemID)
	require.Equal(t, 1, grantedCount)

	require.NoError(t, repo.UpdateStatus(context.Background(), userID, itemID,
		domain.QueueEntryStatusGranted, domain.QueueEntryStatusCancelled))

	grantedCount, usedCount := itemCounters(t, pool, itemID)
	require.Zero(t, grantedCount, "отменённое право обязано вернуть единицу в пул")
	require.Zero(t, usedCount)
}

// TestNeedsReconcile_Predicate — дешёвая проверка перед блокировкой строки
// товара. Предикат обязан покрывать все действия Reconcile: если он пропустит
// случай, соответствующее действие станет недостижимым, и ожидающий зависнет
// в waiting навсегда.
func TestNeedsReconcile_Predicate(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	catalog := NewCatalogRepository(pool)
	ctx := context.Background()

	t.Run("работы нет", func(t *testing.T) {
		itemID := insertTestItem(t, pool, 1, 120)

		need, err := repo.NeedsReconcile(ctx, itemID)
		require.NoError(t, err)
		require.False(t, need, "на пустом товаре продвигать нечего")
	})

	t.Run("есть свободный слот и ожидающий", func(t *testing.T) {
		itemID := insertTestItem(t, pool, 1, 120)
		enterQueue(t, repo, itemID)

		need, err := repo.NeedsReconcile(ctx, itemID)
		require.NoError(t, err)
		require.True(t, need)
	})

	t.Run("права раздали, ждать больше некому", func(t *testing.T) {
		itemID := insertTestItem(t, pool, 1, 120)
		enterQueue(t, repo, itemID)
		grantAll(t, repo, catalog, itemID, 1)

		need, err := repo.NeedsReconcile(ctx, itemID)
		require.NoError(t, err)
		require.False(t, need, "живое право работой для Reconcile не является")
	})

	t.Run("право просрочено", func(t *testing.T) {
		itemID := insertTestItem(t, pool, 1, 120)
		enterQueue(t, repo, itemID)
		userID := grantAll(t, repo, catalog, itemID, 1)[0]

		_, err := pool.Exec(ctx, `
			UPDATE queue_entries SET expires_at = now() - interval '1 second'
			 WHERE item_id = $1 AND user_id = $2`, itemID, userID)
		require.NoError(t, err)

		need, err := repo.NeedsReconcile(ctx, itemID)
		require.NoError(t, err)
		require.True(t, need)
	})

	t.Run("тираж выкуплен, а в очереди ещё ждут", func(t *testing.T) {
		itemID := insertTestItem(t, pool, 1, 120)
		enterQueue(t, repo, itemID)
		buyer := grantAll(t, repo, catalog, itemID, 1)[0]
		enterQueue(t, repo, itemID)

		bought, err := repo.MarkPurchased(ctx, buyer, itemID)
		require.NoError(t, err)
		require.True(t, bought)

		need, err := repo.NeedsReconcile(ctx, itemID)
		require.NoError(t, err)
		require.True(t, need,
			"свободных слотов нет, но ожидающему надо выставить sold_out — иначе он висит вечно")
	})
}

// TestMarkSoldOut_ClosesWaiting — когда тираж разобран, ожидание закрывается
// всем сразу.
func TestMarkSoldOut_ClosesWaiting(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	itemID := insertTestItem(t, pool, 1, 120)
	ctx := context.Background()

	first := enterQueue(t, repo, itemID)
	second := enterQueue(t, repo, itemID)

	require.NoError(t, repo.MarkSoldOut(ctx, itemID))

	require.Equal(t, domain.QueueEntryStatusSoldOut, statusOf(t, pool, first, itemID))
	require.Equal(t, domain.QueueEntryStatusSoldOut, statusOf(t, pool, second, itemID))

	waiting, err := repo.CountByStatus(ctx, itemID, domain.QueueEntryStatusWaiting)
	require.NoError(t, err)
	require.Zero(t, waiting)
}

// TestSnapshot_PositionAndCounters — один запрос, из которого собирается весь
// конверт статуса. Позиция считается по тому же ключу, что и выдача.
func TestSnapshot_PositionAndCounters(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	itemID := insertTestItem(t, pool, 2, 120)
	ctx := context.Background()

	first := enterQueue(t, repo, itemID)
	enterQueue(t, repo, itemID)
	third := enterQueue(t, repo, itemID)

	snapshot, err := repo.Snapshot(ctx, third, itemID)
	require.NoError(t, err)
	require.NotNil(t, snapshot.Entry)
	require.Equal(t, 3, snapshot.Position, "третий вошедший стоит третьим")
	require.Equal(t, 3, snapshot.QueueSize)
	require.Equal(t, 2, snapshot.TotalStock)
	require.False(t, snapshot.ServerTime.IsZero(), "время сервера берётся из Postgres, а не из Go (INV-6)")

	snapshot, err = repo.Snapshot(ctx, first, itemID)
	require.NoError(t, err)
	require.Equal(t, 1, snapshot.Position)

	// Не вставший в очередь — это состояние not_in_queue, а не ошибка (INV-8).
	snapshot, err = repo.Snapshot(ctx, uuid.New(), itemID)
	require.NoError(t, err)
	require.Nil(t, snapshot.Entry)
	require.Equal(t, 3, snapshot.QueueSize)
}

// TestSnapshot_UnknownItem — снимок по несуществующему товару отличается от
// пустой очереди: это 404, а не «никого нет».
func TestSnapshot_UnknownItem(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)

	_, err := repo.Snapshot(context.Background(), uuid.New(), uuid.New())
	require.ErrorIs(t, err, domain.ErrNoItemFound)
}

// TestOutcomes_CountsOnlyFinishedRights — знаменатель конверсии. Права,
// которые ещё держат, в него не входят: их исход неизвестен.
func TestOutcomes_CountsOnlyFinishedRights(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	catalog := NewCatalogRepository(pool)
	itemID := insertTestItem(t, pool, 3, 120)
	ctx := context.Background()

	for range 3 {
		enterQueue(t, repo, itemID)
	}
	granted := grantAll(t, repo, catalog, itemID, 3)

	bought, err := repo.MarkPurchased(ctx, granted[0], itemID)
	require.NoError(t, err)
	require.True(t, bought)

	_, err = pool.Exec(ctx, `
		UPDATE queue_entries SET expires_at = now() - interval '1 second'
		 WHERE item_id = $1 AND user_id = $2`, itemID, granted[1])
	require.NoError(t, err)
	require.NoError(t, repo.InTx(ctx, func(txCtx context.Context) error {
		if _, err := catalog.LockItem(txCtx, itemID); err != nil {
			return err
		}
		_, err := repo.ExpireOverdue(txCtx, itemID)

		return err
	}))

	purchased, expired, err := repo.Outcomes(ctx, itemID)
	require.NoError(t, err)
	require.Equal(t, 1, purchased)
	require.Equal(t, 1, expired)
	require.Equal(t, domain.QueueEntryStatusGranted, statusOf(t, pool, granted[2], itemID),
		"третье право ещё живо и в статистику исходов попасть не должно")
}

// TestGetEntry_ReturnsLatestAttempt — записи участия не удаляются, поэтому
// после повторного входа их несколько, и актуальна самая свежая.
func TestGetEntry_ReturnsLatestAttempt(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	itemID := insertTestItem(t, pool, 1, 120)
	ctx := context.Background()

	userID := enterQueue(t, repo, itemID)
	require.NoError(t, repo.UpdateStatus(ctx, userID, itemID,
		domain.QueueEntryStatusWaiting, domain.QueueEntryStatusCancelled))
	require.NoError(t, repo.Enter(ctx, userID, itemID))

	entry, err := repo.GetEntry(ctx, userID, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueEntryStatusWaiting, entry.Status,
		"актуально последнее участие, а не первая попытка")

	_, err = repo.GetEntry(ctx, uuid.New(), itemID)
	require.ErrorIs(t, err, domain.ErrUserNotFound)
}
