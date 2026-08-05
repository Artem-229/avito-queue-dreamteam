package usecases

import (
	"avito-queue/internal/domain"
	"context"

	"github.com/google/uuid"
)

type purchaseRightRepo interface {
	Create(ctx context.Context, userID, itemID uuid.UUID) error
	GetByUserAndItem(ctx context.Context, userID, itemID uuid.UUID) (domain.PurchaseRight, error)
}

type PurchaseRight struct {
	repo purchaseRightRepo
}

func NewPurchaseRight(repo purchaseRightRepo) *PurchaseRight {
	return &PurchaseRight{
		repo: repo,
	}
}
