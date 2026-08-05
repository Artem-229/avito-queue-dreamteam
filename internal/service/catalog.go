package service

import (
	"context"
	"fmt"

	"avito-queue/internal/domain"
	"avito-queue/internal/repository"
)

type CatalogService interface {
	GetCatalog(ctx context.Context) ([]domain.CatalogItem, error)
	GetCatalogItem(ctx context.Context, id string) (domain.CatalogItem, error)
}

type catalogService struct {
	repo repository.CatalogRepository
}

func NewCatalogService(repo repository.CatalogRepository) CatalogService {
	return &catalogService{
		repo: repo,
	}
}

func (s *catalogService) GetCatalog(ctx context.Context) ([]domain.CatalogItem, error) {
	items, err := s.repo.GetItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("service.GetCatalog: %w", err)
	}

	return items, nil
}

func (s *catalogService) GetCatalogItem(ctx context.Context, id string) (domain.CatalogItem, error) {
	item, err := s.repo.GetItemByID(ctx, id)
	if err != nil {
		return domain.CatalogItem{}, fmt.Errorf("service.GetCatalogItem: %w", err)
	}
	return item, nil
}
