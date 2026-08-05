package app

import (
	"avito-queue/internal/usecases"
	"context"
)

type Usecases struct {
	purchaseRight *usecases.PurchaseRight
}

func NewUsecases(_ context.Context, repositories *Repositories) *Usecases {
	purchaseRightUsecase := usecases.NewPurchaseRight(repositories.PurchaseRightRepo)

	return &Usecases{
		purchaseRight: purchaseRightUsecase,
	}
}
