package app

import (
	"avito-queue/internal/services"
	"context"
	"log/slog"
)

type Services struct {
	PurchaseRight *services.PurchaseRight
	Catalog       *services.CatalogService
	Queue         *services.QueueService
	Checkout      *services.CheckoutService
}

func NewServices(_ context.Context, repositories *Repositories, logger *slog.Logger) *Services {
	purchaseRightService := services.NewPurchaseRight(repositories.PurchaseRightRepo, repositories.QueueRepo)
	catalogService := services.NewCatalogService(repositories.CatalogRepository)
	queueService := services.NewQueueService(repositories.QueueRepo, repositories.PurchaseRightRepo, repositories.CatalogRepository, logger)
	checkoutService := services.NewCheckoutService(repositories.PurchaseRightRepo, purchaseRightService)

	return &Services{
		PurchaseRight: purchaseRightService,
		Catalog:       catalogService,
		Queue:         queueService,
		Checkout:      checkoutService,
	}
}
