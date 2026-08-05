package purchaseright

import (
	"avito-queue/internal/domain"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RightRepo struct {
	pool *pgxpool.Pool
}

func NewRightRepo(pool *pgxpool.Pool) *RightRepo {
	return &RightRepo{
		pool: pool,
	}
}

func (r *RightRepo) Create(ctx context.Context, userID, itemID uuid.UUID) error {
	query := `
		INSERT INTO purchase_rights (user_id, item_id, status, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING`

	t := time.Now()
	res, err := r.pool.Exec(ctx, query, userID, itemID, domain.PurchaseRightStatusGranted, t.Add(5*time.Minute), t)
	if err != nil {
		return fmt.Errorf("creating purchase_right %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("could not insert purchase_right %w", err)
	}

	return nil
}

func (r *RightRepo) GetByUserAndItem(ctx context.Context, userID, itemID uuid.UUID) (domain.PurchaseRight, error) {
	query := `SELECT id, status, user_id, item_id FROM purchase_rights WHERE user_id = $1 AND item_id = $2`

	var right domain.PurchaseRight
	err := r.pool.QueryRow(ctx, query, userID, itemID).Scan(&right.ID, &right.Status, &right.UserID, &right.ItemID)
	if err != nil {
		return right, fmt.Errorf("getting purchase_right %w", err)
	}

	return right, nil
}
