package app

import (
	"avito-queue/internal/clients/gigachat"
	"avito-queue/internal/services"

	"context"
	"log/slog"
	"os"
)

type Services struct {
	QueueEntry *services.QueueEntryService
	Catalog    *services.CatalogService
	Demo       *services.DemoService
	Stats      *services.StatsService
}

func NewServices(_ context.Context, repositories *Repositories, logger *slog.Logger) *Services {
	gigaCfg := gigachat.Config{
		AuthKey:            os.Getenv("GIGACHAT_AUTH_KEY"),
		Scope:              "GIGACHAT_API_PERS",
		InsecureSkipVerify: true,
	}
	gigaClient := gigachat.NewClient(gigaCfg)

	cachedGigaClient := gigachat.NewCachedEmbedder(gigaClient, "/app/configs/embeddings_cache.json")

	chanceCalculator := services.NewChanceCalculator(repositories.QueueEntryRepo)

	catalogService := services.NewCatalogService(repositories.CatalogRepository, cachedGigaClient, repositories.QueueEntryRepo, chanceCalculator)
	queueEntryService := services.NewQueueEntryService(repositories.QueueEntryRepo, repositories.CatalogRepository, chanceCalculator)
	demoService := services.NewDemoService(repositories.QueueEntryRepo, repositories.CatalogRepository, queueEntryService)
	statsService := services.NewStatsService(repositories.StatsRepo)

	return &Services{
		QueueEntry: queueEntryService,
		Catalog:    catalogService,
		Demo:       demoService,
		Stats:      statsService,
	}
}
