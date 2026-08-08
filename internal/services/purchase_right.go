package services

import (
	"avito-queue/internal/domain"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type purchaseRightRepo interface {
	Create(ctx context.Context, userID, itemID uuid.UUID) error
	GetByUserAndItem(ctx context.Context, userID, itemID uuid.UUID) (domain.PurchaseRight, error)
	UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, status domain.PurchaseRightStatus) error
	MarkAsUsed(ctx context.Context, userID, itemID uuid.UUID) (success bool, err error)
	CountUsed(ctx context.Context, itemID uuid.UUID) (int, error)
}

type queueRepo interface {
	UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, status domain.QueueStatus) error
}

type buyCatalogRepo interface {
	GetItemByID(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error)
}

type PurchaseRight struct {
	purchaseRightRepo purchaseRightRepo
	queueRepo         queueRepo
	catalogRepo       buyCatalogRepo
}

func NewPurchaseRight(repo purchaseRightRepo, queueRepo queueRepo, catalogRepo buyCatalogRepo) *PurchaseRight {
	return &PurchaseRight{
		purchaseRightRepo: repo,
		queueRepo:         queueRepo,
		catalogRepo:       catalogRepo,
	}
}

func (p *PurchaseRight) Buy(ctx context.Context, userID, itemID uuid.UUID) error {
	right, err := p.purchaseRightRepo.GetByUserAndItem(ctx, userID, itemID)
	if err != nil {
		if errors.Is(err, domain.ErrNoPurchaseRight) {
			return p.noRightError(ctx, itemID)
		}
		return fmt.Errorf("getting purchase right: %w", err)
	}

	if right.Status != domain.PurchaseRightStatusGranted {
		return p.noRightError(ctx, itemID)
	}

	success, err := p.purchaseRightRepo.MarkAsUsed(ctx, userID, itemID)
	if err != nil {
		return fmt.Errorf("marking purchase right used: %w", err)
	}
	if !success {
		return p.noRightError(ctx, itemID)
	}

	if err := p.queueRepo.UpdateStatus(ctx, userID, itemID, domain.QueueStatusPurchased); err != nil {
		return fmt.Errorf("updating purchase right: %w", err)
	}

	return nil
}

// noRightError уточняет причину отказа в покупке: если весь сток товара уже
// разобран (used >= total_stock), возвращаем domain.ErrItemSoldOut вместо общего
// domain.ErrNoPurchaseRight, чтобы фронт показывал точное сообщение, а не
// обобщённое "нет прав на покупку".
func (p *PurchaseRight) noRightError(ctx context.Context, itemID uuid.UUID) error {
	item, err := p.catalogRepo.GetItemByID(ctx, itemID)
	if err != nil {
		return domain.ErrNoPurchaseRight
	}

	used, err := p.purchaseRightRepo.CountUsed(ctx, itemID)
	if err != nil {
		return domain.ErrNoPurchaseRight
	}

	if used >= item.TotalStock {
		return domain.ErrItemSoldOut
	}

	return domain.ErrNoPurchaseRight
}
