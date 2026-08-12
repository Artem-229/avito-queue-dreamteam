package app

import (
	"avito-queue/internal/clients/gigachat"
	"avito-queue/internal/services"

	"context"
	"log/slog"
	"os"
)

type Services struct {
	PurchaseRight *services.PurchaseRight
	Catalog       *services.CatalogService
	Queue         *services.QueueService
	Demo          *services.DemoService
	Stats         *services.StatsService
}

func NewServices(_ context.Context, repositories *Repositories, logger *slog.Logger) *Services {
	gigaCfg := gigachat.Config{
		AuthKey:            os.Getenv("GIGACHAT_AUTH_KEY"),
		Scope:              "GIGACHAT_API_PERS",
		InsecureSkipVerify: true,
	}
	gigaClient := gigachat.NewClient(gigaCfg)

	catalogService := services.NewCatalogService(repositories.CatalogRepository, gigaClient)

	queueService := services.NewQueueService(repositories.QueueRepo, repositories.PurchaseRightRepo, repositories.CatalogRepository, logger)
	purchaseRightService := services.NewPurchaseRight(repositories.PurchaseRightRepo, repositories.QueueRepo, repositories.CatalogRepository, queueService)
	demoService := services.NewDemoService(repositories.QueueRepo, repositories.CatalogRepository, repositories.PurchaseRightRepo, queueService)
	statsService := services.NewStatsService(repositories.StatsRepo)

	return &Services{
		PurchaseRight: purchaseRightService,
		Catalog:       catalogService,
		Queue:         queueService,
		Demo:          demoService,
		Stats:         statsService,
	}
}
