package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"avito-queue/internal/domain"
)

type CatalogRepository struct {
	pool *pgxpool.Pool
}

func NewCatalogRepository(db *pgxpool.Pool) *CatalogRepository {
	return &CatalogRepository{pool: db}
}

func (r *CatalogRepository) GetItems(ctx context.Context) ([]domain.CatalogItem, error) {
	query := `
		SELECT id, name, price, total_stock, created_at, deleted_at
		FROM catalog_items
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query catalog items: %w", err)
	}
	defer rows.Close()

	var items []domain.CatalogItem
	for rows.Next() {
		var item domain.CatalogItem
		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Price,
			&item.TotalStock,
			&item.CreatedAt,
			&item.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan catalog item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error in catalog items: %w", err)
	}

	return items, nil
}

func (r *CatalogRepository) GetItemByID(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error) {
	query := `
	SELECT id, name, price, total_stock, created_at
	FROM catalog_items
	WHERE id = $1 AND deleted_at IS NULL
	`

	var item domain.CatalogItem
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.Name,
		&item.Price,
		&item.TotalStock,
		&item.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.CatalogItem{}, domain.ErrNoItemFound
		}
		return domain.CatalogItem{}, fmt.Errorf("failed to query item by id: %w", err)
	}
	return item, nil
}
