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
}

type Embedder interface {
	CreateEmbedding(ctx context.Context, text string) ([]float32, error)
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
