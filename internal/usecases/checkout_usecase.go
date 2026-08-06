package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// если есть метод FindByUserAndItem с такими же аргументами и возвращаемыми значениями, ты - репозиторий
type PurchaseRightRepository interface {
	FindByUserAndItem(ctx context.Context, userID, itemID int) (id int, status string, expiresAt time.Time, err error)
}

// поле типизированно, значит можно любой репо закидывать: мок или реальный
type CheckoutUsecase struct {
	repo PurchaseRightRepository
}

//вызываем в мейн, инициализация
func NewCheckoutUsecase(repo PurchaseRightRepository) *CheckoutUsecase {
	return &CheckoutUsecase{repo: repo}
}

func (u *CheckoutUsecase) CheckAccess(ctx context.Context, userID, itemID int) (purchaseID int, allowed bool, reason string, err error) {
	purchaseID, status, expiresAt, err := u.repo.FindByUserAndItem(ctx, userID, itemID)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return 0, false, "Нет права на покупку этого товара", nil
	case err != nil:
		return 0, false, "", err
	case status != "granted" || !expiresAt.After(time.Now()):
		return 0, false, "Право на покупку товара неактивно", nil
	}

	return purchaseID, true, "", nil
}
