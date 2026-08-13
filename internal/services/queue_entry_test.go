package services

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"avito-queue/internal/domain"
)

// TestEntry_ConcurrentNoOversell — главный тест кейса (INV-1): сколько бы
// человек ни ломилось в очередь одновременно, право получают ровно столько,
// сколько единиц в тираже.
//
// Проверяется настоящий QueueEntryService поверх настоящего Postgres: подмена
// репозитория моком выключила бы ровно то, что здесь и проверяется —
// блокировку строки товара и CHECK-констрейнт.
//
// Нарушение констрейнта catalog_items_stock_invariant — это провал теста, а не
// «сработала защита»: значит, логика попыталась раздать больше прав, чем есть
// товара, и последним рубежом оказалась схема.
func TestEntry_ConcurrentNoOversell(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalStock int
		rivals     int
	}{
		{name: "последняя единица на сотню желающих", totalStock: 1, rivals: 100},
		{name: "малый тираж под наплывом", totalStock: 4, rivals: 60},
		{name: "желающих меньше, чем товара", totalStock: 10, rivals: 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pool := testDBPool(t)
			itemID := insertTestItem(t, pool, tt.totalStock, 120)
			service := newTestQueueService(pool)

			statuses := make([]domain.QueueEntryStatus, tt.rivals)
			errs := make([]error, tt.rivals)

			var wg sync.WaitGroup
			wg.Add(tt.rivals)
			for i := range tt.rivals {
				go func() {
					defer wg.Done()

					resp, err := service.Entry(context.Background(), uuid.New(), itemID)
					statuses[i], errs[i] = resp.Status, err
				}()
			}
			wg.Wait()

			for i, err := range errs {
				require.NoErrorf(t, err, "участник %d получил ошибку при входе в очередь", i)
			}

			granted := countEntries(t, pool, itemID, domain.QueueEntryStatusGranted)
			waiting := countEntries(t, pool, itemID, domain.QueueEntryStatusWaiting)
			grantedCount, usedCount := itemCounters(t, pool, itemID)

			expectedGranted := min(tt.totalStock, tt.rivals)
			t.Logf("тираж %d, претендентов %d → выдано прав %d, ждут %d, счётчики товара granted=%d used=%d",
				tt.totalStock, tt.rivals, granted, waiting, grantedCount, usedCount)

			require.Equal(t, expectedGranted, granted,
				"право должны получить ровно min(тираж, претенденты) человек")
			require.Equal(t, tt.rivals-expectedGranted, waiting,
				"все остальные обязаны остаться в очереди, а не потеряться")
			requireCountersMatchEntries(t, pool, itemID, tt.totalStock)

			// Ответ каждого участника обязан совпадать с тем, что в итоге лежит
			// в базе: пользователь не должен увидеть право, которого нет.
			respGranted := 0
			for _, status := range statuses {
				if status == domain.QueueEntryStatusGranted {
					respGranted++
				}
			}
			require.Equal(t, expectedGranted, respGranted,
				"число ответов granted разошлось с числом реально выданных прав")
		})
	}
}

// TestEntry_IsIdempotent — повторный вход не ошибка и не вторая запись:
// пользователь, обновивший страницу, должен увидеть своё же состояние.
func TestEntry_IsIdempotent(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 1, 120)
	service := newTestQueueService(pool)
	ctx := context.Background()
	userID := uuid.New()

	first, err := service.Entry(ctx, userID, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueEntryStatusGranted, first.Status)

	second, err := service.Entry(ctx, userID, itemID)
	require.NoError(t, err, "повторный вход в очередь обязан вернуть состояние, а не ошибку")
	require.Equal(t, first.Status, second.Status)

	var rows int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM queue_entries WHERE item_id = $1 AND user_id = $2`, itemID, userID).Scan(&rows))
	require.Equal(t, 1, rows, "повторный вход не должен заводить вторую запись участия")
	requireCountersMatchEntries(t, pool, itemID, 1)
}

// TestStatus_AdvancesQueueWithoutNewEntries — продвижение очереди ленивое, и
// это осознанное решение вместо фонового воркера. Значит, слот обязан уйти
// следующему от самого опроса статуса: никто новый в очередь не встаёт,
// покупок нет, истекло только время первого.
func TestStatus_AdvancesQueueWithoutNewEntries(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 1, 120)
	service := newTestQueueService(pool)
	ctx := context.Background()

	first, second := uuid.New(), uuid.New()

	firstResp, err := service.Entry(ctx, first, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueEntryStatusGranted, firstResp.Status)

	secondResp, err := service.Entry(ctx, second, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueEntryStatusWaiting, secondResp.Status)
	require.Equal(t, 1, secondResp.Position, "второй в очереди на одну единицу стоит первым на выдачу")

	expireGrants(t, pool, itemID)

	afterExpiry, err := service.Status(ctx, second, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueEntryStatusGranted, afterExpiry.Status,
		"опрос статуса обязан сам погасить просроченное право и отдать слот следующему")
	require.NotNil(t, afterExpiry.ExpiresAt, "у выданного права всегда есть срок")

	require.Equal(t, domain.QueueEntryStatusExpired, statusOf(t, pool, first, itemID),
		"сгоревшее право обязано остаться в истории как expired")
	requireCountersMatchEntries(t, pool, itemID, 1)
}

// TestEntry_ExpiredUserCanRejoin — уникальный индекс по активным статусам не
// должен запирать пользователя навсегда: после сгоревшего права он встаёт
// в очередь заново.
func TestEntry_ExpiredUserCanRejoin(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 1, 120)
	service := newTestQueueService(pool)
	ctx := context.Background()
	userID := uuid.New()

	_, err := service.Entry(ctx, userID, itemID)
	require.NoError(t, err)

	expireGrants(t, pool, itemID)
	require.NoError(t, service.Reconcile(ctx, itemID))
	require.Equal(t, domain.QueueEntryStatusExpired, statusOf(t, pool, userID, itemID))

	again, err := service.Entry(ctx, userID, itemID)
	require.NoError(t, err, "вход после сгоревшего права обязан пройти")
	require.NotErrorIs(t, err, domain.ErrUserAlreadyInQueue)
	require.Equal(t, domain.QueueEntryStatusGranted, again.Status,
		"свободный слот должен вернуться тому, кто встал заново")
	requireCountersMatchEntries(t, pool, itemID, 1)
}

// TestLeave_FromWaitingFreesTheQueue — выход из очереди до выдачи права.
// Тираж не трогается: ожидающий единицу не удерживал.
func TestLeave_FromWaitingFreesTheQueue(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 1, 120)
	service := newTestQueueService(pool)
	ctx := context.Background()

	holder, leaver := uuid.New(), uuid.New()

	_, err := service.Entry(ctx, holder, itemID)
	require.NoError(t, err)
	_, err = service.Entry(ctx, leaver, itemID)
	require.NoError(t, err)

	resp, err := service.Leave(ctx, leaver, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueEntryStatusCancelled, resp.Status)
	require.NotEmpty(t, resp.NextStep.Kind, "у состояния «вы покинули очередь» тоже есть следующий шаг (INV-8)")
	require.Zero(t, countEntries(t, pool, itemID, domain.QueueEntryStatusWaiting),
		"ушедший обязан исчезнуть из очереди ожидания")
	requireCountersMatchEntries(t, pool, itemID, 1)
}

// TestLeave_FromGrantedReturnsSlotToNext — выход держателя права. Слот обязан
// вернуться в пул и уйти следующему сразу, а не при следующем случайном
// опросе кем-нибудь другим.
func TestLeave_FromGrantedReturnsSlotToNext(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 1, 120)
	service := newTestQueueService(pool)
	ctx := context.Background()

	holder, next := uuid.New(), uuid.New()

	holderResp, err := service.Entry(ctx, holder, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueEntryStatusGranted, holderResp.Status)

	_, err = service.Entry(ctx, next, itemID)
	require.NoError(t, err)

	resp, err := service.Leave(ctx, holder, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueEntryStatusCancelled, resp.Status)

	require.Equal(t, domain.QueueEntryStatusGranted, statusOf(t, pool, next, itemID),
		"освободившийся слот обязан уйти следующему в очереди")
	granted, used := itemCounters(t, pool, itemID)
	require.Equal(t, 1, granted, "право одно: ушедшее вернулось в пул и выдано следующему")
	require.Zero(t, used)
	requireCountersMatchEntries(t, pool, itemID, 1)
}

// TestLeave_Rejections — выйти можно только из активного участия. Из
// покупки и из состояния «никогда не вставал» выхода нет, и это разные ошибки:
// у них разные экраны на фронте.
func TestLeave_Rejections(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 1, 120)
	service := newTestQueueService(pool)
	ctx := context.Background()

	buyer := uuid.New()
	_, err := service.Entry(ctx, buyer, itemID)
	require.NoError(t, err)
	require.NoError(t, service.Buy(ctx, buyer, itemID))

	_, err = service.Leave(ctx, buyer, itemID)
	require.ErrorIs(t, err, domain.ErrCannotLeaveQueue,
		"из завершённой покупки выйти нельзя — это терминальное состояние")

	_, err = service.Leave(ctx, uuid.New(), itemID)
	require.ErrorIs(t, err, domain.ErrUserNotFound,
		"выход того, кто в очереди не стоял, — отдельная ошибка, а не «нечего делать»")

	requireCountersMatchEntries(t, pool, itemID, 1)
}

// TestBuy_SpendsRightOnce — покупка тратит право ровно один раз. Повторный
// вызов (обычный клиентский ретрай на плохой сети) не должен превратиться во
// вторую продажу того же слота.
func TestBuy_SpendsRightOnce(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 2, 120)
	service := newTestQueueService(pool)
	ctx := context.Background()
	userID := uuid.New()

	_, err := service.Entry(ctx, userID, itemID)
	require.NoError(t, err)

	require.NoError(t, service.Buy(ctx, userID, itemID))

	granted, used := itemCounters(t, pool, itemID)
	require.Zero(t, granted, "потраченное право перестаёт удерживать единицу тиража")
	require.Equal(t, 1, used)
	require.Equal(t, domain.QueueEntryStatusPurchased, statusOf(t, pool, userID, itemID))

	require.ErrorIs(t, service.Buy(ctx, userID, itemID), domain.ErrNoPurchaseRight,
		"повторная оплата по потраченному праву обязана быть отклонена")
	requireCountersMatchEntries(t, pool, itemID, 2)
}

// TestBuy_WithoutRightIsRejected — обход очереди: прямой запрос на покупку в
// обход экрана ожидания. Ожидающий правом ещё не владеет, посторонний — тем
// более.
func TestBuy_WithoutRightIsRejected(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 1, 120)
	service := newTestQueueService(pool)
	ctx := context.Background()

	holder, waiter, stranger := uuid.New(), uuid.New(), uuid.New()

	_, err := service.Entry(ctx, holder, itemID)
	require.NoError(t, err)
	_, err = service.Entry(ctx, waiter, itemID)
	require.NoError(t, err)

	require.ErrorIs(t, service.Buy(ctx, waiter, itemID), domain.ErrNoPurchaseRight,
		"стоящий в очереди не может купить в обход выдачи права")
	require.ErrorIs(t, service.Buy(ctx, stranger, itemID), domain.ErrNoPurchaseRight,
		"не вставший в очередь не может купить вовсе")

	require.Equal(t, domain.QueueEntryStatusGranted, statusOf(t, pool, holder, itemID),
		"чужие попытки покупки не должны задевать держателя права")
	requireCountersMatchEntries(t, pool, itemID, 1)
}

// TestBuy_ExpiredRightIsRejected — право, истёкшее за миллисекунду до оплаты,
// покупкой не становится. Проверка срока живёт в самом UPDATE, поэтому окна
// между «посмотрели» и «потратили» не существует.
func TestBuy_ExpiredRightIsRejected(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 1, 120)
	service := newTestQueueService(pool)
	ctx := context.Background()
	userID := uuid.New()

	_, err := service.Entry(ctx, userID, itemID)
	require.NoError(t, err)
	expireGrants(t, pool, itemID)

	require.ErrorIs(t, service.Buy(ctx, userID, itemID), domain.ErrNoPurchaseRight)
	require.Equal(t, domain.QueueEntryStatusExpired, statusOf(t, pool, userID, itemID))

	granted, used := itemCounters(t, pool, itemID)
	require.Zero(t, granted, "сгоревшее право обязано вернуть единицу в пул")
	require.Zero(t, used)
}

// TestBuy_SoldOutIsDistinctFromNoRight — когда тираж разобран, пользователю
// нужно сказать «товар закончился», а не «у вас нет права»: это разные экраны
// и разные следующие шаги.
func TestBuy_SoldOutIsDistinctFromNoRight(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 1, 120)
	service := newTestQueueService(pool)
	ctx := context.Background()

	buyer := uuid.New()
	_, err := service.Entry(ctx, buyer, itemID)
	require.NoError(t, err)
	require.NoError(t, service.Buy(ctx, buyer, itemID))

	require.ErrorIs(t, service.Buy(ctx, uuid.New(), itemID), domain.ErrItemSoldOut,
		"на выкупленном тираже причина отказа — распроданный товар, а не отсутствие права")
}

// TestBuy_ConcurrentSameRight — двадцать одновременных нажатий «оплатить» по
// одному праву. Успех обязан быть ровно один: проверка права и его трата —
// один UPDATE, между ними окна нет.
func TestBuy_ConcurrentSameRight(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 1, 120)
	service := newTestQueueService(pool)
	ctx := context.Background()
	userID := uuid.New()

	_, err := service.Entry(ctx, userID, itemID)
	require.NoError(t, err)

	const attempts = 20
	errs := make([]error, attempts)

	var wg sync.WaitGroup
	wg.Add(attempts)
	for i := range attempts {
		go func() {
			defer wg.Done()
			errs[i] = service.Buy(context.Background(), userID, itemID)
		}()
	}
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		// Отказ различает причину: пока тираж не выкуплен — «нет права», после
		// выкупа последней единицы — «товар закончился». Обе формулировки
		// корректны, потому что гонка решается в произвольный момент.
		require.Truef(t, errorIsAny(err, domain.ErrNoPurchaseRight, domain.ErrItemSoldOut),
			"проигравшая гонку попытка обязана получить доменный отказ, а не внутреннюю ошибку: %v", err)
	}

	t.Logf("%d одновременных оплат по одному праву → успешных покупок: %d", attempts, succeeded)
	require.Equal(t, 1, succeeded, "одно право — одна покупка")
	requireCountersMatchEntries(t, pool, itemID, 1)
}

// TestQueueLifecycle_ConcurrentEntryAndPurchase — сквозной сценарий кейса под
// конкуренцией: толпа входит в очередь, все получившие право одновременно
// оплачивают, остальные упираются в распроданный тираж.
//
// Здесь сходятся все три операции, меняющие распределение (Entry, Buy,
// Reconcile), поэтому тест и стоит последним: он ловит расхождения, которые по
// отдельности не видны.
func TestQueueLifecycle_ConcurrentEntryAndPurchase(t *testing.T) {
	t.Parallel()

	const (
		totalStock = 3
		rivals     = 30
	)

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, totalStock, 120)
	service := newTestQueueService(pool)

	users := make([]uuid.UUID, rivals)
	for i := range users {
		users[i] = uuid.New()
	}

	// Ошибки собираются и проверяются в основной горутине: require из
	// побочной завершил бы её, а не тест, и падение выглядело бы как зависание.
	entryErrs := make([]error, rivals)

	var wg sync.WaitGroup
	wg.Add(rivals)
	for i, userID := range users {
		go func() {
			defer wg.Done()
			_, entryErrs[i] = service.Entry(context.Background(), userID, itemID)
		}()
	}
	wg.Wait()

	for i, err := range entryErrs {
		require.NoErrorf(t, err, "участник %d получил ошибку при входе в очередь", i)
	}
	require.Equal(t, totalStock, countEntries(t, pool, itemID, domain.QueueEntryStatusGranted))

	// Оплачивают все разом — и держатели прав, и те, кто ещё ждёт.
	buyErrs := make([]error, rivals)
	wg.Add(rivals)
	for i, userID := range users {
		go func() {
			defer wg.Done()
			buyErrs[i] = service.Buy(context.Background(), userID, itemID)
		}()
	}
	wg.Wait()

	purchases := 0
	for _, err := range buyErrs {
		if err == nil {
			purchases++
			continue
		}
		require.Truef(t,
			errorIsAny(err, domain.ErrNoPurchaseRight, domain.ErrItemSoldOut),
			"отказ обязан быть доменным (нет права либо товар кончился), получено: %v", err)
	}

	granted, used := itemCounters(t, pool, itemID)
	t.Logf("тираж %d, претендентов %d → покупок %d, счётчики товара granted=%d used=%d",
		totalStock, rivals, purchases, granted, used)

	require.Equal(t, totalStock, purchases, "продано ровно столько единиц, сколько было в тираже")
	requireCountersMatchEntries(t, pool, itemID, totalStock)

	// Тираж выкуплен: ожидающим больше нечего ждать, и они обязаны узнать об
	// этом с альтернативами, а не остаться в waiting навсегда.
	//
	// Проигравший ищется по результату покупки, а не по позиции в срезе:
	// входили все одновременно, и порядок FIFO задаёт база, а не порядок,
	// в котором тест раздавал идентификаторы.
	var loser uuid.UUID
	for i, err := range buyErrs {
		if err != nil {
			loser = users[i]
			break
		}
	}
	require.NotEqual(t, uuid.Nil, loser, "при 30 претендентах на 3 единицы кто-то обязан остаться без покупки")

	resp, err := service.Status(context.Background(), loser, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueEntryStatusSoldOut, resp.Status)
	require.NotEmpty(t, resp.Message)
	require.Equal(t, "browse_similar", resp.NextStep.Kind,
		"следующий шаг на распроданном товаре — уйти к похожим лотам, а не в тупик")
	require.NotNil(t, resp.Alternatives, "alternatives всегда массив, даже пустой")
	require.Zero(t, countEntries(t, pool, itemID, domain.QueueEntryStatusWaiting),
		"после выкупа тиража в очереди не должно остаться ожидающих")
}

func errorIsAny(err error, targets ...error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}

	return false
}
