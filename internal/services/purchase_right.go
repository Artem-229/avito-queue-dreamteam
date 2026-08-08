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
	MarkAsUsed(ctx context.Context, userID, itemID uuid.UUID) (success bool, err error)
}

type queueRepo interface {
	UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, status domain.QueueStatus) error
}

type PurchaseRight struct {
	purchaseRightRepo purchaseRightRepo
	queueRepo         queueRepo
}

func NewPurchaseRight(repo purchaseRightRepo, queueRepo queueRepo) *PurchaseRight {
	return &PurchaseRight{
		purchaseRightRepo: repo,
		queueRepo:         queueRepo,
	}
}

func (p *PurchaseRight) Buy(ctx context.Context, userID, itemID uuid.UUID) error {
	right, err := p.purchaseRightRepo.GetByUserAndItem(ctx, userID, itemID)
	if err != nil {
		return fmt.Errorf("getting purchase right: %w", err)
	}

	if right.Status != domain.PurchaseRightStatusGranted {
		return domain.ErrNoPurchaseRight
	}

	success, err := p.purchaseRightRepo.MarkAsUsed(ctx, userID, itemID)
	if err != nil {
		return fmt.Errorf("marking purchase right used: %w", err)
	}
	if !success {
		return domain.ErrNoPurchaseRight
	}

	if err := p.queueRepo.UpdateStatus(ctx, userID, itemID, domain.QueueStatusPurchased); err != nil {
		return fmt.Errorf("updating purchase right: %w", err)
	}

	return nil
}
