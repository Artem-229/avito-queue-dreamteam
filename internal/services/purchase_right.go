package services

import (
	"avito-queue/internal/domain"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type purchaseRightRepo interface {
	GetByUserAndItem(ctx context.Context, userID, itemID uuid.UUID) (domain.PurchaseRight, error)
	MarkAsUsed(ctx context.Context, userID, itemID uuid.UUID) (success bool, err error)
	CountUsed(ctx context.Context, itemID uuid.UUID) (int, error)
}

type queueRepo interface {
	UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, from, to domain.QueueStatus) error
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type buyCatalogRepo interface {
	GetItemByID(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error)
	LockItem(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error)
	AdjustCounts(ctx context.Context, id uuid.UUID, grantedDelta, usedDelta int) error
}

// queueAdvancer продвигает очередь перед покупкой: без этого Buy мог бы
// увидеть уже истёкшее, но ещё не погашенное право.
type queueAdvancer interface {
	EnsureAdvanced(ctx context.Context, itemID uuid.UUID) error
}

type PurchaseRight struct {
	purchaseRightRepo purchaseRightRepo
	queueRepo         queueRepo
	catalogRepo       buyCatalogRepo
	queue             queueAdvancer
}

func NewPurchaseRight(repo purchaseRightRepo, queueRepo queueRepo, catalogRepo buyCatalogRepo, queue queueAdvancer) *PurchaseRight {
	return &PurchaseRight{
		purchaseRightRepo: repo,
		queueRepo:         queueRepo,
		catalogRepo:       catalogRepo,
		queue:             queue,
	}
}

// Buy тратит выданное право — вместе с Reconcile единственное место, меняющее
// распределение прав (INV-2). Тот же порядок блокировок: LockItem первым
// запросом транзакции (INV-3), правки в одной транзакции (INV-5).
func (p *PurchaseRight) Buy(ctx context.Context, userID, itemID uuid.UUID) error {
	if err := p.queue.EnsureAdvanced(ctx, itemID); err != nil {
		return fmt.Errorf("ensuring queue advanced: %w", err)
	}

	return p.queueRepo.InTx(ctx, func(ctx context.Context) error {
		item, err := p.catalogRepo.LockItem(ctx, itemID)
		if err != nil {
			return fmt.Errorf("locking item: %w", err)
		}

		right, err := p.purchaseRightRepo.GetByUserAndItem(ctx, userID, itemID)
		if err != nil {
			if errors.Is(err, domain.ErrNoPurchaseRight) {
				return p.noRightError(ctx, item)
			}
			return fmt.Errorf("getting purchase right: %w", err)
		}

		if right.Status != domain.PurchaseRightStatusGranted {
			return p.noRightError(ctx, item)
		}

		success, err := p.purchaseRightRepo.MarkAsUsed(ctx, userID, itemID)
		if err != nil {
			return fmt.Errorf("marking purchase right used: %w", err)
		}
		if !success {
			return p.noRightError(ctx, item)
		}

		if err := p.queueRepo.UpdateStatus(ctx, userID, itemID,
			domain.QueueStatusGranted, domain.QueueStatusPurchased); err != nil {
			return fmt.Errorf("updating queue record: %w", err)
		}

		if err := p.catalogRepo.AdjustCounts(ctx, itemID, -1, 1); err != nil {
			return fmt.Errorf("adjusting stock counters: %w", err)
		}

		return nil
	})
}

// noRightError уточняет причину отказа: если весь сток разобран, возвращает
// ErrItemSoldOut вместо общего ErrNoPurchaseRight.
func (p *PurchaseRight) noRightError(ctx context.Context, item domain.CatalogItem) error {
	used, err := p.purchaseRightRepo.CountUsed(ctx, item.ID)
	if err != nil {
		return domain.ErrNoPurchaseRight
	}

	if used >= item.TotalStock {
		return domain.ErrItemSoldOut
	}

	return domain.ErrNoPurchaseRight
}
