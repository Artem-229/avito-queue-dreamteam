package services

import (
	"context"
	"fmt"

	"avito-queue/internal/domain"

	"github.com/google/uuid"
)

type CatalogRepository interface {
	GetItems(ctx context.Context) ([]domain.CatalogItem, error)
	GetItemByID(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error)
	GetSimilarItems(ctx context.Context, item domain.CatalogItem) ([]domain.CatalogItem, error)
	SaveEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error
	GetItemsWithoutEmbeddings(ctx context.Context) ([]domain.CatalogItem, error)
}

type Embedder interface {
	CreateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// queueCounter — размер очереди по товару.
type queueCounter interface {
	CountByStatus(ctx context.Context, itemID uuid.UUID, status domain.QueueEntryStatus) (int, error)
}

type CatalogService struct {
	repo     CatalogRepository
	embedder Embedder
	queue    queueCounter
	chance   chanceCalculator
}

func NewCatalogService(repo CatalogRepository, embedder Embedder, queue queueCounter, chance chanceCalculator) *CatalogService {
	return &CatalogService{
		repo:     repo,
		embedder: embedder,
		queue:    queue,
		chance:   chance,
	}
}

func (s *CatalogService) GetCatalog(ctx context.Context) ([]domain.CatalogItem, error) {
	items, err := s.repo.GetItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("services.GetCatalog: %w", err)
	}

	return items, nil
}

// GetCatalogItem — карточка товара вместе с оценкой шанса до входа в очередь.
func (s *CatalogService) GetCatalogItem(ctx context.Context, id uuid.UUID) (domain.CatalogItemCard, error) {
	item, err := s.repo.GetItemByID(ctx, id)
	if err != nil {
		return domain.CatalogItemCard{}, fmt.Errorf("services.GetCatalogItem: %w", err)
	}

	queueSize, err := s.queue.CountByStatus(ctx, id, domain.QueueEntryStatusWaiting)
	if err != nil {
		return domain.CatalogItemCard{}, fmt.Errorf("services.GetCatalogItem: counting queue: %w", err)
	}

	card := domain.CatalogItemCard{CatalogItem: item, QueueSize: queueSize}

	if item.UsedCount >= item.TotalStock {
		return card, nil
	}

	percent, basis, err := s.chance.For(ctx, id, queueSize+1, item.GrantedCount, item.UsedCount, item.TotalStock)
	if err != nil {
		return domain.CatalogItemCard{}, fmt.Errorf("services.GetCatalogItem: calculating chance: %w", err)
	}

	card.ChanceIfJoin = &domain.PurchaseChance{Percent: percent, Basis: basis}

	return card, nil
}

func (s *CatalogService) GetSimilarItems(ctx context.Context, id uuid.UUID) ([]domain.CatalogItem, error) {
	item, err := s.repo.GetItemByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("services.GetItemByID: %w", err)
	}

	if len(item.Embedding) == 0 {
		textToEmbed := fmt.Sprintf("Категория: %s. Название: %s", item.Category, item.Name)

		emb, err := s.embedder.CreateEmbedding(ctx, textToEmbed)
		if err != nil {
			return nil, fmt.Errorf("failed to create embedding via GigaChat: %w", err)
		}

		if err := s.repo.SaveEmbedding(ctx, item.ID, emb); err != nil {
			return nil, fmt.Errorf("failed to save embedding: %w", err)
		}

		item.Embedding = emb
	}

	items, err := s.repo.GetSimilarItems(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("services.GetSimilarItems: %w", err)
	}

	return items, nil
}

func (s *CatalogService) WarmupEmbeddings(ctx context.Context) {
	items, err := s.repo.GetItemsWithoutEmbeddings(ctx)
	if err != nil {
		fmt.Printf("Ошибка при поиске товаров для прогрева: %v\n", err)
		return
	}

	if len(items) == 0 {
		fmt.Println("Прогрев эмбеддингов не требуется: у всех товаров есть векторы.")
		return
	}

	fmt.Printf("Начинаем прогрев эмбеддингов для %d товаров...\n", len(items))

	for _, item := range items {
		textToEmbed := fmt.Sprintf("Категория: %s. Название: %s", item.Category, item.Name)

		emb, err := s.embedder.CreateEmbedding(ctx, textToEmbed)
		if err != nil {
			fmt.Printf("Ошибка векторизации товара %s: %v\n", item.ID, err)
			continue // Если с одним товаром беда — идем дальше
		}

		err = s.repo.SaveEmbedding(ctx, item.ID, emb)
		if err != nil {
			fmt.Printf("Ошибка сохранения вектора для %s: %v\n", item.ID, err)
		}
	}

	fmt.Println("Прогрев эмбеддингов успешно завершен!")
}
