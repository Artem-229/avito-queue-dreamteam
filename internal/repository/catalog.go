package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"avito-queue/internal/domain"
)

type CatalogRepository interface {
	GetItems(ctx context.Context) ([]entity.CatalogItem, error)
	GetItemByID(ctx context.Context, id string) (entity.CatalogItem, error)
}

type catalogRepo struct {
	db *pgxpool.Pool
}

func NewCatalogRepository(db *pgxpool.Pool) CatalogRepository {
	return &catalogRepo{db: db}
}

func (r *catalogRepo) GetItems(ctx context.Context) ([]entity.CatalogItem, error) {
	query := `
		SELECT id, name, price, total_stock, created_at, deleted_at
		FROM catalog_items
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query catalog items: %w", err)
	}
	defer rows.Close()

	var items []entity.CatalogItem
	for rows.Next() {
		var item entity.CatalogItem
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

func (r *catalogRepo) GetItemByID(ctx context.Context, id string) (entity.CatalogItem, error) {
	query := `
	SELECT id, name, price, total_stock, created_at, deleted_at
	FROM catalog_items
	WHERE id = $1 AND deleted_at IS NULL
	`

	var item entity.CatalogItem
	err := r.db.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.Name,
		&item.Price,
		&item.TotalStock,
		&item.CreatedAt,
		&item.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return entity.CatalogItem{}, fmt.Errorf("item not found")
		}
		return entity.CatalogItem{}, fmt.Errorf("failed to query item by id: %w", err)
	}
	return item, nil
}
