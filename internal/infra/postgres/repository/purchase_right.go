package repository

import (
	"avito-queue/internal/domain"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PurchaseRightRepository struct {
	pool *pgxpool.Pool
}

func NewPurchaseRightRepo(pool *pgxpool.Pool) *PurchaseRightRepository {
	return &PurchaseRightRepository{
		pool: pool,
	}
}

func (r *PurchaseRightRepository) Create(ctx context.Context, userID, itemID uuid.UUID) error {
	query := `
		INSERT INTO purchase_rights (user_id, item_id, status, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, item_id) WHERE status = 'granted' DO NOTHING`

	t := time.Now()
	res, err := r.pool.Exec(ctx, query, userID, itemID, domain.PurchaseRightStatusGranted, t.Add(5*time.Minute), t)
	if err != nil {
		return fmt.Errorf("creating purchase_right %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("could not insert purchase_right")
	}

	return nil
}

func (r *PurchaseRightRepository) GetByUserAndItem(ctx context.Context, userID, itemID uuid.UUID) (domain.PurchaseRight, error) {
	query := `
		SELECT id, status, user_id, item_id, expires_at
		FROM purchase_rights
		WHERE user_id = $1 AND item_id = $2
		ORDER BY created_at DESC
		LIMIT 1`

	var right domain.PurchaseRight
	err := r.pool.QueryRow(ctx, query, userID, itemID).Scan(&right.ID, &right.Status, &right.UserID, &right.ItemID, &right.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PurchaseRight{}, domain.ErrNoPurchaseRight
		}
		return domain.PurchaseRight{}, fmt.Errorf("getting purchase_right: %w", err)
	}

	return right, nil
}

func (r *PurchaseRightRepository) ExpireOld(ctx context.Context, itemID uuid.UUID, t time.Time) ([]uuid.UUID, error) {
	query := `
			UPDATE purchase_rights 
			SET status = $1 WHERE item_id = $2
			AND status = $3
			AND expires_at < $4
			RETURNING user_id`

	var IDs []uuid.UUID
	rows, err := r.pool.Query(ctx, query, domain.PurchaseRightStatusExpired, itemID, domain.PurchaseRightStatusGranted, t)
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
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterating thought rows %w", rows.Err())
	}

	return IDs, nil
}

func (r *PurchaseRightRepository) UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, status domain.PurchaseRightStatus) error {
	query := `UPDATE purchase_rights SET status = $1 WHERE user_id = $2 AND item_id = $3`
	_, err := r.pool.Exec(ctx, query, status, userID, itemID)
	if err != nil {
		return fmt.Errorf("update purchase_right status: %w", err)
	}
	return nil
}

func (r *PurchaseRightRepository) CountGranted(ctx context.Context, itemID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM purchase_rights WHERE item_id = $1 AND status = $2`
	var count int
	err := r.pool.QueryRow(ctx, query, itemID, domain.PurchaseRightStatusGranted).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting purchase_right %w", err)
	}

	return count, nil
}

func (r *PurchaseRightRepository) CountUsed(ctx context.Context, itemID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM purchase_rights WHERE item_id = $1 AND status = $2`
	var count int
	err := r.pool.QueryRow(ctx, query, itemID, domain.PurchaseRightStatusUsed).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting purchase_right %w", err)
	}

	return count, nil
}

func (r *PurchaseRightRepository) MarkAsUsed(ctx context.Context, userID, itemID uuid.UUID) (success bool, err error) {
	query := `
		UPDATE purchase_rights SET status = $3
		WHERE user_id = $1 AND item_id = $2 AND status = $4 AND expires_at > now()`

	tag, err := r.pool.Exec(ctx, query, userID, itemID, domain.PurchaseRightStatusUsed, domain.PurchaseRightStatusGranted)
	if err != nil {
		return false, err
	}

	if tag.RowsAffected() == 0 {
		return false, nil
	}
	return true, nil
}
