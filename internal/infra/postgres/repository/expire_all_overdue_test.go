package repository

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"avito-queue/internal/domain"
)

// TestExpireAllOverdue_ConcurrentWithReconcile_NoDeadlock — доказывает то,
// что предыдущая версия теста не доказывала: реальное пересечение по
// времени двух путей гашения (глобального ExpireAllOverdue и точечного
// Reconcile-пути через LockItem+ExpireOverdue) на одном товаре не приводит
// ни к дедлоку (SQLSTATE 40P01), ни к двойному декременту granted_count.
func TestExpireAllOverdue_ConcurrentWithReconcile_NoDeadlock(t *testing.T) {
	pool := testDBPool(t)
	repo := NewQueueEntryRepo(pool)
	catalog := NewCatalogRepository(pool)

	itemID := insertTestItem(t, pool, 10, 120)

	// Реальный путь: встать в очередь, получить право через настоящий
	// GrantNext (не руками через SQL) — так счётчики товара согласованы с
	// тем, что реально сделал бы прод-код, а не подделаны вручную.
	const overdueCount = 5
	users := make([]uuid.UUID, 0, overdueCount)
	for i := 0; i < overdueCount; i++ {
		users = append(users, enterQueue(t, repo, itemID))
	}
	granted := grantAll(t, repo, catalog, itemID, overdueCount)
	require.Len(t, granted, overdueCount, "все ожидающие должны получить право — тираж позволяет")

	// Права реально выданы (expires_at в будущем) — искусственно сдвигаем
	// срок в прошлое, тем же приёмом, что и демо-ручка ExpireNow, вместо
	// того чтобы ждать hold_ttl_seconds в реальном времени.
	_, err := pool.Exec(context.Background(),
		`UPDATE queue_entries SET expires_at = now() - interval '1 second'
		 WHERE item_id = $1 AND status = $2`,
		itemID, domain.QueueEntryStatusGranted)
	require.NoError(t, err, "push expires_at into the past")

	grantedBefore, _ := itemCounters(t, pool, itemID)
	require.Equal(t, overdueCount, grantedBefore, "granted_count must reflect the real grants before expiring anything")

	// Барьер в два шага: сначала обе горутины подтверждают, что реально
	// дошли до точки старта и заблокировались (readyWg), и только когда обе
	// готовы — открывается канал start. Без этого шага одна горутина могла
	// бы не успеть дойти до <-start к моменту close(start) и просто не
	// притормозить вообще — из-за этого предыдущая версия теста иногда
	// проходила вхолостую, ничего не проверяя.
	start := make(chan struct{})
	var readyWg sync.WaitGroup
	readyWg.Add(2)

	var wg sync.WaitGroup
	var reconcileErr, globalErr error
	wg.Add(2)

	go func() {
		defer wg.Done()
		readyWg.Done()
		<-start
		reconcileErr = repo.InTx(context.Background(), func(ctx context.Context) error {
			if _, err := catalog.LockItem(ctx, itemID); err != nil {
				return err
			}
			_, err := repo.ExpireOverdue(ctx, itemID)
			return err
		})
	}()

	go func() {
		defer wg.Done()
		readyWg.Done()
		<-start
		globalErr = repo.ExpireAllOverdue(context.Background())
	}()

	readyWg.Wait() // ждём, пока ОБЕ горутины реально дойдут до <-start
	close(start)   // и только теперь отпускаем их одновременно
	wg.Wait()

	require.NoError(t, reconcileErr, "Reconcile-путь не должен падать (дедлока быть не должно)")
	require.NoError(t, globalErr, "ExpireAllOverdue не должен падать (дедлока быть не должно)")

	grantedAfter, usedAfter := itemCounters(t, pool, itemID)
	require.Equal(t, 0, grantedAfter,
		"granted_count должен стать ровно 0 — отрицательное значение означало бы двойной декремент, положительное — что часть прав не погасилась")
	require.Equal(t, 0, usedAfter, "expired не должен трогать used_count")

	for _, userID := range users {
		require.Equal(t, domain.QueueEntryStatusExpired, statusOf(t, pool, userID, itemID),
			"каждое из пяти просроченных прав должно погаситься ровно один раз")
	}
}

// TestExpireAllOverdue_ConcurrentWithReconcile_ManyRounds — тот же сценарий
// много раз подряд: единичный проход мог случайно не столкнуться с окном
// гонки даже с барьером (зависит от того, как ОС планирует горутины/сетевые
// round-trip к Postgres). Повтор снижает шанс ложного зелёного результата.
func TestExpireAllOverdue_ConcurrentWithReconcile_ManyRounds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping repeated race test in short mode")
	}

	for round := 0; round < 20; round++ {
		t.Run("", func(t *testing.T) {
			TestExpireAllOverdue_ConcurrentWithReconcile_NoDeadlock(t)
		})
	}
}
