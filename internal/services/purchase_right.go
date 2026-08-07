package services

import (
	"avito-queue/internal/domain"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type purchaseRightRepo interface {
	Create(ctx context.Context, userID, itemID uuid.UUID) error
	GetByUserAndItem(ctx context.Context, userID, itemID uuid.UUID) (domain.PurchaseRight, error)
	UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, status domain.PurchaseRightStatus) error
}

type PurchaseRight struct {
	repo purchaseRightRepo
}

func NewPurchaseRight(repo purchaseRightRepo) *PurchaseRight {
	return &PurchaseRight{
		repo: repo,
	}
}

func (p *PurchaseRight) Buy(ctx context.Context, userID, itemID uuid.UUID) error {
	right, err := p.repo.GetByUserAndItem(ctx, userID, itemID)
	if err != nil {
		return fmt.Errorf("getting purchase right: %w", err)
	}

	if right.Status != domain.PurchaseRightStatusGranted {
		return domain.ErrNoPurchaseRight
	}

	if err := p.repo.UpdateStatus(ctx, userID, itemID, domain.PurchaseRightStatusUsed); err != nil {
		return fmt.Errorf("marking purchase right used: %w", err)
	}

	return nil
}
