package services

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"avito-queue/internal/domain"
)

// TestReconcile_NoOversell — A-04 / INF-06: главный тест кейса (CLAUDE.md §3.4,
// TASKS-REMAINING.md Т-12). Товар с total_stock=1, 100 горутин параллельно
// вызывают QueueService.Entry (реальный сервис поверх реального Postgres) для
// 100 разных пользователей. Ожидание: ровно одно активное право, и
// queue.status='granted', и purchase_rights.status='granted' согласны друг с
// другом (INV-5), granted_count=1, used_count=0 (INV-1). Единственная
// допустимая ошибка при конкуренции — domain.ErrUserAlreadyInQueue; ошибка
// констрейнта catalog_items_stock_invariant — это провал теста: значит логика
// попыталась продать товар дважды, а не то, что "сработала защита".
func TestReconcile_NoOversell(t *testing.T) {
	pool := testDBPool(t)
	itemID := insertCatalogItem(t, pool, 1, 120)
	svc := newTestQueueService(pool)

	const n = 100
	var wg sync.WaitGroup
	errs := make([]error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := svc.Entry(context.Background(), uuid.New(), itemID)
			errs[i] = err
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		if err == nil {
			continue
		}
		require.NotContains(t, err.Error(), "catalog_items_stock_invariant",
			"constraint violation means the logic tried to break INV-1, a caught defect, not a working safeguard: %v", err)
		require.ErrorIsf(t, err, domain.ErrUserAlreadyInQueue,
			"only domain.ErrUserAlreadyInQueue is an acceptable concurrency error, got: %v", err)
	}

	grantedQueue := countQueueByStatus(t, pool, itemID, domain.QueueStatusGranted)
	grantedRights := countPurchaseRightsByStatus(t, pool, itemID, domain.PurchaseRightStatusGranted)
	require.Equal(t, 1, grantedQueue, "exactly one queue record must end up granted")
	require.Equal(t, 1, grantedRights, "exactly one purchase right must end up granted")
	require.Equal(t, grantedQueue, grantedRights,
		"queue.status='granted' count must equal purchase_rights.status='granted' count (INV-5)")

	grantedCount, usedCount := catalogCounts(t, pool, itemID)
	require.Equal(t, 1, grantedCount, "granted_count must reflect exactly the one right handed out")
	require.Equal(t, 0, usedCount)
}

// TestReconcile_ExpiredUserCanRejoin — D-02: до фикса MarkGrantedExpired искал
// status='waiting' и никогда не находил строк, поэтому пользователь с
// протухшим правом навсегда запирался в очереди (UNIQUE (user_id, item_id)
// WHERE status IN ('waiting','granted') не давал войти заново). Право
// истекает не через time.Sleep, а прямой записью expires_at в прошлое в БД.
func TestReconcile_ExpiredUserCanRejoin(t *testing.T) {
	pool := testDBPool(t)
	itemID := insertCatalogItem(t, pool, 1, 120)
	svc := newTestQueueService(pool)
	ctx := context.Background()

	userA := uuid.New()
	resp, err := svc.Entry(ctx, userA, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueStatusGranted, resp.Status, "sole entrant on a stock=1 item must be granted immediately")

	_, err = pool.Exec(ctx,
		`UPDATE purchase_rights SET expires_at = now() - interval '1 hour' WHERE item_id = $1 AND user_id = $2 AND status = $3`,
		itemID, userA, domain.PurchaseRightStatusGranted)
	require.NoError(t, err, "forcing expiry directly in the DB, not via time.Sleep")

	require.NoError(t, svc.Reconcile(ctx, itemID))

	require.Equal(t, domain.QueueStatusExpired, mustGetQueueStatus(t, pool, userA, itemID),
		"queue.status must flip to expired once the granted right expired (D-02)")

	resp2, err := svc.Entry(ctx, userA, itemID)
	require.NoError(t, err, "rejoining after expiry must succeed, not return ErrUserAlreadyInQueue")
	require.NotErrorIs(t, err, domain.ErrUserAlreadyInQueue)
	require.Equal(t, domain.QueueStatusGranted, resp2.Status, "the freed slot must be handed straight back to the rejoining user")
}

// TestSlotGoesToNext_WithoutNewEntries — D-03: сценарий, который жюри
// проверит руками первым. Один товар, двое в очереди: первый получает право
// и не покупает, право протухает, и **никто новый в очередь не встаёт**.
// Продвижение должно случиться от самого опроса статуса вторым пользователем
// (GetStatus -> EnsureAdvanced -> Reconcile), а не только из Entry.
func TestSlotGoesToNext_WithoutNewEntries(t *testing.T) {
	pool := testDBPool(t)
	itemID := insertCatalogItem(t, pool, 1, 120)
	svc := newTestQueueService(pool)
	ctx := context.Background()

	userA := uuid.New()
	userB := uuid.New()

	respA, err := svc.Entry(ctx, userA, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueStatusGranted, respA.Status)

	respB, err := svc.Entry(ctx, userB, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueStatusWaiting, respB.Status)

	_, err = pool.Exec(ctx,
		`UPDATE purchase_rights SET expires_at = now() - interval '1 hour' WHERE item_id = $1 AND user_id = $2 AND status = $3`,
		itemID, userA, domain.PurchaseRightStatusGranted)
	require.NoError(t, err)

	// Никто новый в очередь не встаёт — единственное действие теста это опрос
	// статуса тем, кто уже стоял вторым.
	statusResp, err := svc.GetStatus(ctx, userB, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueStatusGranted, statusResp.Status,
		"slot must go to the next waiter purely from a status poll, without any new entrant joining (D-03)")
	require.NotNil(t, statusResp.ExpiresAt)

	require.Equal(t, domain.QueueStatusExpired, mustGetQueueStatus(t, pool, userA, itemID))
	require.Equal(t, domain.QueueStatusGranted, mustGetQueueStatus(t, pool, userB, itemID))
}

// TestSlotGoesToNext_OnLeave — T-11/D-12, тот же исход что и
// TestSlotGoesToNext_WithoutNewEntries, но с другим триггером: не истечение
// права, а явный выход из очереди (Leave). Leave обязана погасить активное
// право и декрементировать granted_count в той же транзакции, что и
// queue.status (INV-5), и Reconcile сразу после коммита должен отдать
// освободившийся слот следующему.
func TestSlotGoesToNext_OnLeave(t *testing.T) {
	pool := testDBPool(t)
	itemID := insertCatalogItem(t, pool, 1, 120)
	svc := newTestQueueService(pool)
	ctx := context.Background()

	userA := uuid.New()
	userB := uuid.New()

	respA, err := svc.Entry(ctx, userA, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueStatusGranted, respA.Status)

	respB, err := svc.Entry(ctx, userB, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueStatusWaiting, respB.Status)

	leaveResp, err := svc.Leave(ctx, userA, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueStatusCancelled, leaveResp.Status,
		"Leave must return the same status envelope as GetStatus")

	require.Equal(t, domain.QueueStatusCancelled, mustGetQueueStatus(t, pool, userA, itemID))
	require.Equal(t, domain.QueueStatusGranted, mustGetQueueStatus(t, pool, userB, itemID),
		"the slot released by Leave must go to the next waiter without waiting for a poll")

	require.Equal(t, 1, countPurchaseRightsByStatus(t, pool, itemID, domain.PurchaseRightStatusCancelled),
		"leaving with an active right must cancel that right, not leave it dangling as granted")
	require.Equal(t, 1, countPurchaseRightsByStatus(t, pool, itemID, domain.PurchaseRightStatusGranted))

	grantedCount, _ := catalogCounts(t, pool, itemID)
	require.Equal(t, 1, grantedCount, "granted_count nets back to 1: -1 for A's cancelled right, +1 for B's new one")
}

// TestQueueRightsConsistency — INV-5/D-14: право хранится дублированно, в
// purchase_rights.status и в queue.status, синхронизируется только вручную
// внутри одной транзакции. Гоняется после конкурентного сценария: число
// queue.status='granted' должно совпасть с числом
// purchase_rights.status='granted' для того же товара.
func TestQueueRightsConsistency(t *testing.T) {
	pool := testDBPool(t)
	itemID := insertCatalogItem(t, pool, 3, 120)
	svc := newTestQueueService(pool)

	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, _ = svc.Entry(context.Background(), uuid.New(), itemID)
		}()
	}
	wg.Wait()

	grantedQueue := countQueueByStatus(t, pool, itemID, domain.QueueStatusGranted)
	grantedRights := countPurchaseRightsByStatus(t, pool, itemID, domain.PurchaseRightStatusGranted)
	require.Equal(t, grantedQueue, grantedRights,
		"queue.status='granted' count must equal purchase_rights.status='granted' count for the same item (INV-5/D-14)")
	require.Equal(t, 3, grantedRights, "stock=3 must grant exactly 3 rights out of 20 contenders")
}

// TestBuy_ConcurrentPurchase_SingleSuccess — дополнительный тест к
// TestMarkAsUsed_Concurrency (internal/infra/postgres/repository), но на
// уровне сервиса PurchaseRight.Buy: он же трогает AdjustCounts и
// queue.status в той же транзакции (INV-5), которые repo-level тест не видит.
// N горутин бьются за одно и то же уже выданное право, ожидание: ровно один
// успех, единственная допустимая ошибка при конкуренции —
// domain.ErrItemSoldOut (весь сток уже использован конкурентом), никаких
// нарушений catalog_items_stock_invariant.
func TestBuy_ConcurrentPurchase_SingleSuccess(t *testing.T) {
	pool := testDBPool(t)
	itemID := insertCatalogItem(t, pool, 1, 120)
	queueSvc := newTestQueueService(pool)
	buySvc := newTestPurchaseRight(pool, queueSvc)
	ctx := context.Background()

	userID := uuid.New()
	resp, err := queueSvc.Entry(ctx, userID, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueStatusGranted, resp.Status)

	const n = 100
	var wg sync.WaitGroup
	errs := make([]error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			errs[i] = buySvc.Buy(ctx, userID, itemID)
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, err := range errs {
		if err == nil {
			successCount++
			continue
		}
		require.NotContains(t, err.Error(), "catalog_items_stock_invariant",
			"constraint violation here means Buy tried to sell an already-used right again: %v", err)
		require.ErrorIsf(t, err, domain.ErrItemSoldOut,
			"only domain.ErrItemSoldOut is an acceptable concurrency error for Buy, got: %v", err)
	}
	require.Equal(t, 1, successCount, "exactly one of the concurrent Buy calls must succeed")

	require.Equal(t, 1, countPurchaseRightsByStatus(t, pool, itemID, domain.PurchaseRightStatusUsed))
	require.Equal(t, domain.QueueStatusPurchased, mustGetQueueStatus(t, pool, userID, itemID))

	grantedCount, usedCount := catalogCounts(t, pool, itemID)
	require.Equal(t, 0, grantedCount, "AdjustCounts must move the right out of granted on purchase")
	require.Equal(t, 1, usedCount)
}

// TestSoldOut_AfterStockExhausted — A-09. Регрессия на рассинхрон сторожа и
// действия: MarkSoldOut живёт внутри Reconcile, а NeedsReconcile пускал туда
// только при наличии свободного слота. При выкупленном стоке свободных слотов
// ноль по определению, поэтому ожидающий оставался в waiting навсегда вместо
// sold_out. Проверяем через GetStatus, а не прямым Reconcile: именно GetStatus
// поллит фронт, и именно он обязан сам подтянуть состояние.
func TestSoldOut_AfterStockExhausted(t *testing.T) {
	pool := testDBPool(t)
	itemID := insertCatalogItem(t, pool, 1, 120)
	queueSvc := newTestQueueService(pool)
	buySvc := newTestPurchaseRight(pool, queueSvc)
	ctx := context.Background()

	buyer, waiter := uuid.New(), uuid.New()

	granted, err := queueSvc.Entry(ctx, buyer, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueStatusGranted, granted.Status)

	waiting, err := queueSvc.Entry(ctx, waiter, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueStatusWaiting, waiting.Status)

	require.NoError(t, buySvc.Buy(ctx, buyer, itemID))

	resp, err := queueSvc.GetStatus(ctx, waiter, itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueStatusSoldOut, resp.Status,
		"waiter must see sold_out once the whole stock is used, not stay in waiting forever")
	require.NotEmpty(t, resp.Message, "sold_out must carry a user-facing message (INV-8)")
	require.NotEmpty(t, resp.NextStep.Kind, "sold_out must carry a next step (INV-8)")
}

// TestSoldOut_ZeroStockItem — тот же дефект в самой заметной форме: у товара с
// total_stock = 0 свободных слотов нет с самого начала, поэтому старый предикат
// не пускал Reconcile ни разу и вход в очередь отдавал waiting. Такой товар
// есть в демо-данных (migrations/00004), то есть сценарий воспроизводится на
// стенде руками.
func TestSoldOut_ZeroStockItem(t *testing.T) {
	pool := testDBPool(t)
	itemID := insertCatalogItem(t, pool, 0, 120)
	queueSvc := newTestQueueService(pool)

	resp, err := queueSvc.Entry(context.Background(), uuid.New(), itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueStatusSoldOut, resp.Status,
		"joining a queue for an item with zero stock must resolve to sold_out immediately")
}
