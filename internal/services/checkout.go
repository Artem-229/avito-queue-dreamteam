package services

import (
	"avito-queue/internal/domain"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type PurchaseRightReader interface {
	GetByUserAndItem(ctx context.Context, userID, itemID uuid.UUID) (domain.PurchaseRight, error)
}

type PurchaseRightBuyer interface {
	Buy(ctx context.Context, userID, itemID uuid.UUID) error
}

type CheckoutService struct {
	repo          PurchaseRightReader
	purchaseRight PurchaseRightBuyer
}

func NewCheckoutService(repo PurchaseRightReader, purchaseRight PurchaseRightBuyer) *CheckoutService {
	return &CheckoutService{repo: repo, purchaseRight: purchaseRight}
}

func (u *CheckoutService) CheckAccess(ctx context.Context, userID, itemID uuid.UUID) (purchaseID uuid.UUID, expiresAt time.Time, allowed bool, reason string, err error) {
	purchaseRight, err := u.repo.GetByUserAndItem(ctx, userID, itemID)

	switch {
	case errors.Is(err, domain.ErrNoPurchaseRight):
		return uuid.Nil, time.Time{}, false, "Нет права на покупку этого товара", nil
	case err != nil:
		return uuid.Nil, time.Time{}, false, "", err
	case purchaseRight.Status != domain.PurchaseRightStatusGranted || !purchaseRight.ExpiresAt.After(time.Now()):
		return uuid.Nil, time.Time{}, false, "Право на покупку товара неактивно", nil
	}

	return purchaseRight.ID, purchaseRight.ExpiresAt, true, "", nil
}

func (u *CheckoutService) Pay(ctx context.Context, userID, itemID uuid.UUID) (success bool, reason string, err error) {
	err = u.purchaseRight.Buy(ctx, userID, itemID)

	switch {
	case errors.Is(err, domain.ErrItemSoldOut):
		return false, "товар полностью распродан", nil
	case errors.Is(err, domain.ErrNoPurchaseRight):
		return false, "право на покупку недействительно, истекло или уже использовано", nil
	case err != nil:
		return false, "", err
	}

	return true, "", nil
}
