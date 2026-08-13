package services

import (
	"context"
	"fmt"
	"time"

	"avito-queue/internal/domain"

	"github.com/google/uuid"
)

type CatalogRepository interface {
	GetItems(ctx context.Context) ([]domain.CatalogItem, error)
	GetItemByID(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error)
	GetSimilarItems(ctx context.Context, id uuid.UUID) ([]domain.CatalogItem, error)
	SaveEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error
	GetItemsWithoutEmbeddings(ctx context.Context) ([]domain.CatalogItem, error)

	CreateItem(ctx context.Context, item domain.CatalogItem, embedding []float32) error
	UpdateItem(ctx context.Context, item domain.CatalogItem) error
	DeleteItem(ctx context.Context, id uuid.UUID) error
}

type Embedder interface {
	CreateEmbedding(ctx context.Context, text string) ([]float32, error)
	Invalidate(text string)
}

type CatalogService struct {
	repo     CatalogRepository
	embedder Embedder
}

func NewCatalogService(repo CatalogRepository, embedder Embedder) *CatalogService {
	return &CatalogService{
		repo:     repo,
		embedder: embedder,
	}
}

func (s *CatalogService) GetCatalog(ctx context.Context) ([]domain.CatalogItem, error) {
	items, err := s.repo.GetItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("services.GetCatalog: %w", err)
	}

	return items, nil
}

func (s *CatalogService) GetCatalogItem(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error) {
	item, err := s.repo.GetItemByID(ctx, id)
	if err != nil {
		return domain.CatalogItem{}, fmt.Errorf("services.GetCatalogItem: %w", err)
	}
	return item, nil
}

func (s *CatalogService) GetSimilarItems(ctx context.Context, id uuid.UUID) ([]domain.CatalogItem, error) {
	item, err := s.repo.GetItemByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("services.GetItemByID: %w", err)
	}

	if !item.HasEmbedding {
		textToEmbed := fmt.Sprintf("Категория: %s. Название: %s", item.Category, item.Name)

		emb, err := s.embedder.CreateEmbedding(ctx, textToEmbed)
		if err != nil {
			return nil, fmt.Errorf("failed to create embedding via GigaChat: %w", err)
		}

		if err := s.repo.SaveEmbedding(ctx, item.ID, emb); err != nil {
			return nil, fmt.Errorf("failed to save embedding: %w", err)
		}
	}

	items, err := s.repo.GetSimilarItems(ctx, id)
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
			continue
		}

		err = s.repo.SaveEmbedding(ctx, item.ID, emb)
		if err != nil {
			fmt.Printf("Ошибка сохранения вектора для %s: %v\n", item.ID, err)
		}
	}

	fmt.Println("Прогрев эмбеддингов успешно завершен!")
}

func (s *CatalogService) CreateItem(ctx context.Context, dto domain.CatalogItemCreateDTO) (domain.CatalogItem, error) {
	item := domain.CatalogItem{
		ID:             uuid.New(),
		Name:           dto.Name,
		PriceKopecks:   dto.PriceKopecks,
		TotalStock:     dto.TotalStock,
		HoldTTLSeconds: dto.HoldTTLSeconds,
		Category:       dto.Category,
		SellerName:     dto.SellerName,
		CreatedAt:      time.Now().UTC(),
	}

	var emb []float32
	textToEmbed := fmt.Sprintf("Категория: %s. Название: %s", item.Category, item.Name)
	if createdEmb, err := s.embedder.CreateEmbedding(ctx, textToEmbed); err == nil {
		emb = createdEmb
		item.HasEmbedding = true
	}

	if err := s.repo.CreateItem(ctx, item, emb); err != nil {
		return domain.CatalogItem{}, fmt.Errorf("services.CreateItem: %w", err)
	}

	return item, nil
}

func (s *CatalogService) PatchItem(ctx context.Context, id uuid.UUID, dto domain.CatalogItemPatchDTO) (domain.CatalogItem, error) {
	item, err := s.repo.GetItemByID(ctx, id)
	if err != nil {
		return domain.CatalogItem{}, fmt.Errorf("services.PatchItem (GetItemByID): %w", err)
	}

	needsNewEmbedding := false
	oldText := fmt.Sprintf("Категория: %s. Название: %s", item.Category, item.Name)

	if dto.Name != nil && *dto.Name != item.Name {
		item.Name = *dto.Name
		needsNewEmbedding = true
	}
	if dto.Category != nil && *dto.Category != item.Category {
		item.Category = *dto.Category
		needsNewEmbedding = true
	}
	if dto.PriceKopecks != nil {
		item.PriceKopecks = *dto.PriceKopecks
	}
	if dto.TotalStock != nil {
		item.TotalStock = *dto.TotalStock
	}
	if dto.HoldTTLSeconds != nil {
		item.HoldTTLSeconds = *dto.HoldTTLSeconds
	}
	if dto.SellerName != nil {
		item.SellerName = *dto.SellerName
	}

	if needsNewEmbedding {
		newText := fmt.Sprintf("Категория: %s. Название: %s", item.Category, item.Name)
		if emb, err := s.embedder.CreateEmbedding(ctx, newText); err == nil {
			_ = s.repo.SaveEmbedding(ctx, item.ID, emb)
			s.embedder.Invalidate(oldText)
			item.HasEmbedding = true
		}
	}

	if err := s.repo.UpdateItem(ctx, item); err != nil {
		return domain.CatalogItem{}, fmt.Errorf("services.PatchItem (UpdateItem): %w", err)
	}

	return item, nil
}

func (s *CatalogService) DeleteItem(ctx context.Context, id uuid.UUID) error {
	item, err := s.repo.GetItemByID(ctx, id)
	if err == nil {
		s.embedder.Invalidate(fmt.Sprintf("Категория: %s. Название: %s", item.Category, item.Name))
	}
	return s.repo.DeleteItem(ctx, id)
}
