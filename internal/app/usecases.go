package app

import (
	"avito-queue/internal/services"
	"context"
)

type Services struct {
	PurchaseRight *services.PurchaseRight
	Catalog       *services.CatalogService
	Queue         *services.Queue
}

func NewServices(_ context.Context, repositories *Repositories) *Services {
	purchaseRightService := services.NewPurchaseRight(repositories.PurchaseRightRepo)
	catalogService := services.NewCatalogService(repositories.CatalogRepository)
	queueService := services.NewQueueService(repositories.QueueRepo, repositories.PurchaseRightRepo, repositories.CatalogRepository)

	return &Services{
		PurchaseRight: purchaseRightService,
		Catalog:       catalogService,
		Queue:         queueService,
	}
}
