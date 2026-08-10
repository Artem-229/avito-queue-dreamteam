package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"avito-queue/internal/domain"
)

// mockAdvanceQueueRepo — минимальная реализация QueueRepo для проверки
// EnsureAdvanced/NeedsReconcile без сети (T-09, D-03). Только InTx и
// NeedsReconcile участвуют в поведении, которое проверяет тест; остальные
// методы — заглушки, чтобы удовлетворить интерфейс.
type mockAdvanceQueueRepo struct {
	needsReconcile    bool
	needsReconcileErr error
	inTxCalled        bool
}

func (m *mockAdvanceQueueRepo) Entry(ctx context.Context, userID, itemID uuid.UUID) error {
	return nil
}

func (m *mockAdvanceQueueRepo) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	m.inTxCalled = true
	return fn(ctx)
}

func (m *mockAdvanceQueueRepo) MarkGrantedExpired(ctx context.Context, itemID uuid.UUID, userIDs []uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockAdvanceQueueRepo) NeedsReconcile(ctx context.Context, itemID uuid.UUID) (bool, error) {
	return m.needsReconcile, m.needsReconcileErr
}

func (m *mockAdvanceQueueRepo) GetWaiting(ctx context.Context, itemID uuid.UUID, freeSlots int) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *mockAdvanceQueueRepo) UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, from, to domain.QueueStatus) error {
	return nil
}

func (m *mockAdvanceQueueRepo) GetRecord(ctx context.Context, userID, itemID uuid.UUID) (domain.Queue, error) {
	return domain.Queue{}, nil
}

func (m *mockAdvanceQueueRepo) GetPosition(ctx context.Context, queueRecord domain.Queue) (int, error) {
	return 0, nil
}

func (m *mockAdvanceQueueRepo) MarkSoldOut(ctx context.Context, itemID uuid.UUID) error {
	return nil
}

func (m *mockAdvanceQueueRepo) CountWaiting(ctx context.Context, itemID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockAdvanceQueueRepo) Now(ctx context.Context) (time.Time, error) {
	return time.Time{}, nil
}

// mockAdvanceCatalogRepo — фиксирует, был ли вызван LockItem: он вызывается
// только внутри Reconcile (первым запросом транзакции, INV-3), поэтому
// lockItemCalled — прямой индикатор того, что Reconcile реально запускался.
type mockAdvanceCatalogRepo struct {
	lockItemCalled bool
}

func (m *mockAdvanceCatalogRepo) GetItemByID(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error) {
	return domain.CatalogItem{TotalStock: 1}, nil
}

func (m *mockAdvanceCatalogRepo) LockItem(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error) {
	m.lockItemCalled = true
	return domain.CatalogItem{TotalStock: 1}, nil
}

func (m *mockAdvanceCatalogRepo) AdjustCounts(ctx context.Context, id uuid.UUID, grantedDelta, usedDelta int) error {
	return nil
}

func (m *mockAdvanceCatalogRepo) GetSimilarItems(ctx context.Context, item domain.CatalogItem) ([]domain.CatalogItem, error) {
	return nil, nil
}

type mockAdvancePurchaseRightRepo struct{}

func (m *mockAdvancePurchaseRightRepo) Create(ctx context.Context, userID, itemID uuid.UUID) error {
	return nil
}

func (m *mockAdvancePurchaseRightRepo) CountGranted(ctx context.Context, itemID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockAdvancePurchaseRightRepo) CountUsed(ctx context.Context, itemID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockAdvancePurchaseRightRepo) UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, from, to domain.PurchaseRightStatus) error {
	return nil
}

func (m *mockAdvancePurchaseRightRepo) ExpireOld(ctx context.Context, itemID uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *mockAdvancePurchaseRightRepo) GetByUserAndItem(ctx context.Context, userID, itemID uuid.UUID) (domain.PurchaseRight, error) {
	return domain.PurchaseRight{}, nil
}

// TestEnsureAdvanced_SkipsReconcileWhenNotNeeded — T-09/D-03: NeedsReconcile
// — дешёвая проверка без блокировки товара специально для того, чтобы частый
// поллинг статуса не превращал взятие блокировки в горячую точку (см.
// комментарий у EnsureAdvanced в services/queue.go). Если NeedsReconcile
// говорит "работы нет", Reconcile (а значит и QueueRepo.InTx, и
// CatalogRepo.LockItem) не должен вызываться вовсе.
func TestEnsureAdvanced_SkipsReconcileWhenNotNeeded(t *testing.T) {
	qr := &mockAdvanceQueueRepo{needsReconcile: false}
	cr := &mockAdvanceCatalogRepo{}
	pr := &mockAdvancePurchaseRightRepo{}
	svc := NewQueueService(qr, pr, cr, discardLogger())

	err := svc.EnsureAdvanced(context.Background(), uuid.New())

	require.NoError(t, err)
	require.False(t, qr.inTxCalled, "Reconcile's transaction must not open when NeedsReconcile=false")
	require.False(t, cr.lockItemCalled, "LockItem must not be taken when there is no work to reconcile")
}

// TestEnsureAdvanced_RunsReconcileWhenNeeded — обратная сторона того же
// контракта: когда NeedsReconcile сигналит, что есть просроченные права или
// одновременно свободные слоты и ожидающие, EnsureAdvanced обязана
// действительно продвинуть очередь (запустить Reconcile), а не просто
// прочитать состояние.
func TestEnsureAdvanced_RunsReconcileWhenNeeded(t *testing.T) {
	qr := &mockAdvanceQueueRepo{needsReconcile: true}
	cr := &mockAdvanceCatalogRepo{}
	pr := &mockAdvancePurchaseRightRepo{}
	svc := NewQueueService(qr, pr, cr, discardLogger())

	err := svc.EnsureAdvanced(context.Background(), uuid.New())

	require.NoError(t, err)
	require.True(t, qr.inTxCalled, "Reconcile must open its transaction when NeedsReconcile=true")
	require.True(t, cr.lockItemCalled, "Reconcile must take the item lock as its first statement (INV-3)")
}
