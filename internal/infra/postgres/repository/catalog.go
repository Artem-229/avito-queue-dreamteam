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
		SELECT id, name, price_kopecks, total_stock, hold_ttl_seconds, granted_count, used_count, category, seller_name, created_at
		FROM catalog_items
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := db(ctx, r.pool).Query(ctx, query)
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
			&item.PriceKopecks,
			&item.TotalStock,
			&item.HoldTTLSeconds,
			&item.GrantedCount,
			&item.UsedCount,
			&item.Category,
			&item.SellerName,
			&item.CreatedAt,
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
	SELECT id, name, price_kopecks, total_stock, hold_ttl_seconds, granted_count, used_count, category, seller_name, created_at
	FROM catalog_items
	WHERE id = $1 AND deleted_at IS NULL
	`

	var item domain.CatalogItem
	err := db(ctx, r.pool).QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.Name,
		&item.PriceKopecks,
		&item.TotalStock,
		&item.HoldTTLSeconds,
		&item.GrantedCount,
		&item.UsedCount,
		&item.Category,
		&item.SellerName,
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

// GetSimilarItems — лоты той же категории, у которых ещё остались невыкупленные
// единицы: предлагать распроданный товар взамен распроданного бессмысленно.
// Удержанные права (granted_count) в фильтр не входят — они могут сгореть,
// и слот вернётся.
func (r *CatalogRepository) GetSimilarItems(ctx context.Context, item domain.CatalogItem) ([]domain.CatalogItem, error) {
	query := `
		SELECT id, name, price_kopecks, total_stock, hold_ttl_seconds, granted_count, used_count, category, seller_name, created_at
		FROM catalog_items
		WHERE category = $1 AND id != $2 AND deleted_at IS NULL
		  AND used_count < total_stock
		ORDER BY created_at, id
		LIMIT 20`

	rows, err := db(ctx, r.pool).Query(ctx, query, item.Category, item.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog items: %w", err)
	}
	defer rows.Close()

	var items []domain.CatalogItem
	for rows.Next() {
		var item domain.CatalogItem
		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.PriceKopecks,
			&item.TotalStock,
			&item.HoldTTLSeconds,
			&item.GrantedCount,
			&item.UsedCount,
			&item.Category,
			&item.SellerName,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan catalog item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error in similar catalog items: %w", err)
	}

	return items, nil
}

func (r *CatalogRepository) LockItem(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error) {
	query := `
		SELECT id, name, price_kopecks, total_stock, hold_ttl_seconds, granted_count, used_count, category, seller_name, created_at
		FROM catalog_items
		WHERE id = $1 AND deleted_at IS NULL
		FOR NO KEY UPDATE`

	if _, ok := extractTx(ctx); !ok {
		return domain.CatalogItem{}, fmt.Errorf("LockItem вне транзакции: %w", domain.ErrNoTx)
	}

	var item domain.CatalogItem
	err := db(ctx, r.pool).QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.Name,
		&item.PriceKopecks,
		&item.TotalStock,
		&item.HoldTTLSeconds,
		&item.GrantedCount,
		&item.UsedCount,
		&item.Category,
		&item.SellerName,
		&item.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.CatalogItem{}, domain.ErrNoItemFound
		}
		return domain.CatalogItem{}, fmt.Errorf("locking catalog item: %w", err)
	}
	return item, nil
}

func (r *CatalogRepository) AdjustCounts(ctx context.Context, id uuid.UUID, grantedDelta, usedDelta int) error {
	if grantedDelta == 0 && usedDelta == 0 {
		return nil
	}

	query := `
		UPDATE catalog_items
		   SET granted_count = granted_count + $2,
		       used_count    = used_count + $3
		 WHERE id = $1`

	tag, err := db(ctx, r.pool).Exec(ctx, query, id, grantedDelta, usedDelta)
	if err != nil {
		return fmt.Errorf("adjusting catalog item counts: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNoItemFound
	}

	return nil
}
