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
}

type CatalogService struct {
	repo CatalogRepository
}

func NewCatalogService(repo CatalogRepository) *CatalogService {
	return &CatalogService{
		repo: repo,
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
		return nil, fmt.Errorf("services.GetCatalogItem: %w", err)
	}

	items, err := s.repo.GetSimilarItems(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("services.GetSimilarItems: %w", err)
	}

	return items, nil
}
