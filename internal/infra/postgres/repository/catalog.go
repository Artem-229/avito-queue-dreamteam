package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

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

func (r *CatalogRepository) SaveEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error {
	query := `UPDATE catalog_items SET embedding = $1 WHERE id = $2`

	_, err := db(ctx, r.pool).Exec(ctx, query, pgvector.NewVector(embedding), id)
	if err != nil {
		return fmt.Errorf("failed to save embedding: %w", err)
	}
	return nil
}

func (r *CatalogRepository) GetSimilarItems(ctx context.Context, item domain.CatalogItem) ([]domain.CatalogItem, error) {

	if len(item.Embedding) == 0 {
		return []domain.CatalogItem{}, nil
	}

	query := `
		SELECT id, name, price_kopecks, total_stock, hold_ttl_seconds, granted_count, used_count, category, seller_name, created_at
		FROM catalog_items
		WHERE id != $1 
		  AND deleted_at IS NULL
		  AND used_count < total_stock
		  AND embedding IS NOT NULL
		ORDER BY embedding <=> $2
		LIMIT 5`

	rows, err := db(ctx, r.pool).Query(ctx, query, item.ID, pgvector.NewVector(item.Embedding))
	if err != nil {
		return nil, fmt.Errorf("failed to get similar items via vector: %w", err)
	}
	defer rows.Close()

	var items []domain.CatalogItem
	for rows.Next() {
		var i domain.CatalogItem
		err := rows.Scan(
			&i.ID, &i.Name, &i.PriceKopecks, &i.TotalStock,
			&i.HoldTTLSeconds, &i.GrantedCount, &i.UsedCount,
			&i.Category, &i.SellerName, &i.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan similar item: %w", err)
		}
		items = append(items, i)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error in similar catalog items: %w", err)
	}

	return items, nil
}

func (r *CatalogRepository) GetItemsWithoutEmbeddings(ctx context.Context) ([]domain.CatalogItem, error) {
	query := `
		SELECT id, name, price_kopecks, total_stock, hold_ttl_seconds, granted_count, used_count, category, seller_name, created_at
		FROM catalog_items
		WHERE deleted_at IS NULL AND embedding IS NULL
	`

	rows, err := db(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query items without embeddings: %w", err)
	}
	defer rows.Close()

	var items []domain.CatalogItem
	for rows.Next() {
		var item domain.CatalogItem
		err := rows.Scan(
			&item.ID, &item.Name, &item.PriceKopecks, &item.TotalStock,
			&item.HoldTTLSeconds, &item.GrantedCount, &item.UsedCount,
			&item.Category, &item.SellerName, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
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

func (r *CatalogRepository) CreateItem(ctx context.Context, item domain.CatalogItem) error {
	query := `
		INSERT INTO catalog_items (id, name, price_kopecks, total_stock, hold_ttl_seconds, granted_count, used_count, category, seller_name, created_at, embedding)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	var emb pgvector.Vector
	if len(item.Embedding) > 0 {
		emb = pgvector.NewVector(item.Embedding)
	}

	_, err := db(ctx, r.pool).Exec(ctx, query,
		item.ID, item.Name, item.PriceKopecks, item.TotalStock,
		item.HoldTTLSeconds, item.GrantedCount, item.UsedCount,
		item.Category, item.SellerName, item.CreatedAt, emb,
	)
	return err
}

func (r *CatalogRepository) UpdateItem(ctx context.Context, item domain.CatalogItem) error {
	query := `
		UPDATE catalog_items 
		SET name = $2, price_kopecks = $3, total_stock = $4, hold_ttl_seconds = $5, 
		    category = $6, seller_name = $7, embedding = $8
		WHERE id = $1 AND deleted_at IS NULL
	`
	var emb interface{}
	if len(item.Embedding) > 0 {
		emb = pgvector.NewVector(item.Embedding)
	}

	_, err := db(ctx, r.pool).Exec(ctx, query,
		item.ID, item.Name, item.PriceKopecks, item.TotalStock,
		item.HoldTTLSeconds, item.Category, item.SellerName, emb,
	)
	return err
}

func (r *CatalogRepository) DeleteItem(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE catalog_items SET deleted_at = NOW() WHERE id = $1`
	tag, err := db(ctx, r.pool).Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNoItemFound
	}
	return nil
}
