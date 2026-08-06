package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PurchaseRightRepository interface {
	FindByUserAndItem(ctx context.Context, userID, itemID uuid.UUID) (id uuid.UUID, status string, expiresAt time.Time, err error)
	MarkAsUsed(ctx context.Context, purchaseID uuid.UUID) (success bool, err error)
}

type CheckoutUsecase struct {
	repo PurchaseRightRepository
}

func NewCheckoutUsecase(repo PurchaseRightRepository) *CheckoutUsecase {
	return &CheckoutUsecase{repo: repo}
}

func (u *CheckoutUsecase) CheckAccess(ctx context.Context, userID, itemID uuid.UUID) (purchaseID uuid.UUID, expiresAt time.Time, allowed bool, reason string, err error) {
	purchaseID, status, expiresAt, err := u.repo.FindByUserAndItem(ctx, userID, itemID)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return uuid.Nil, time.Time{}, false, "Нет права на покупку этого товара", nil
	case err != nil:
		return uuid.Nil, time.Time{}, false, "", err
	case status != "granted" || !expiresAt.After(time.Now()):
		return uuid.Nil, time.Time{}, false, "Право на покупку товара неактивно", nil
	}

	return purchaseID, expiresAt, true, "", nil
}

func (u *CheckoutUsecase) Pay(ctx context.Context, purchaseID uuid.UUID) (success bool, reason string, err error) {
	success, err = u.repo.MarkAsUsed(ctx, purchaseID)
	if err != nil {
		return false, "", err
	}

	if !success {
		return false, "право на покупку недействительно или уже использовано", nil
	}

	return true, "", nil
}
