package repository

import (
	"avito-queue/internal/domain"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PurchaseRightRepo struct {
	pool *pgxpool.Pool
}

func NewPurchaseRightRepo(pool *pgxpool.Pool) *PurchaseRightRepo {
	return &PurchaseRightRepo{
		pool: pool,
	}
}

func (r *PurchaseRightRepo) Create(ctx context.Context, userID, itemID uuid.UUID) error {
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

func (r *PurchaseRightRepo) GetByUserAndItem(ctx context.Context, userID, itemID uuid.UUID) (domain.PurchaseRight, error) {
	query := `SELECT id, status, user_id, item_id FROM purchase_rights WHERE user_id = $1 AND item_id = $2`

	var right domain.PurchaseRight
	err := r.pool.QueryRow(ctx, query, userID, itemID).Scan(&right.ID, &right.Status, &right.UserID, &right.ItemID)
	if err != nil {
		return right, fmt.Errorf("getting purchase_right %w", err)
	}

	return right, nil
}

func (r *PurchaseRightRepo) ExpireOld(ctx context.Context, itemID uuid.UUID, t time.Time) ([]uuid.UUID, error) {
	query := `
			UPDATE purchase_rights 
			SET status = $1 WHERE item_id = $2
			ANS status = $3
			AND expires_at < $4
			RETURNING user_id`

	var IDs []uuid.UUID
	rows, err := r.pool.Query(ctx, query, domain.PurchaseRightStatusExpired, domain.PurchaseRightStatusGranted, itemID, t)
	if err != nil {
		return IDs, fmt.Errorf("getting purchase_right %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ID uuid.UUID
		if err := rows.Scan(&ID); err != nil {
			return IDs, fmt.Errorf("scanning purchase_right %w", err)
		}
		IDs = append(IDs, ID)
	}

	return IDs, nil
}

func (r *PurchaseRightRepo) UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, status domain.PurchaseRightStatus) error {
	query := `UPDATE purchase_rights SET status = $1 WHERE user_id = $2 AND item_id = $3`
	_, err := r.pool.Exec(ctx, query, status, userID, itemID)
	if err != nil {
		return fmt.Errorf("update purchase_right status: %w", err)
	}
	return nil
}

func (r *PurchaseRightRepo) CountActive(ctx context.Context, itemID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM purchase_rights WHERE item_id = $1 AND status = $2`
	var count int
	err := r.pool.QueryRow(ctx, query, itemID, domain.PurchaseRightStatusGranted).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting purchase_right %w", err)
	}

	return count, nil
}
