package repository

import (
	"avito-queue/internal/domain"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PurchaseRightRepo struct {
	pool *pgxpool.Pool
}

func NewPurchaseRightRepo(pool *pgxpool.Pool) *PurchaseRightRepo {
	return &PurchaseRightRepo{pool: pool}
}

const findPurchaseRightQuery = `
	SELECT id, status, expires_at
	FROM purchase_rights
	WHERE user_id = $1 AND item_id = $2`

func (r *PurchaseRightRepo) FindByUserAndItem(ctx context.Context, userID, itemID uuid.UUID) (id uuid.UUID, status domain.PurchaseStatus, expiresAt time.Time, err error) {
	err = r.pool.QueryRow(ctx, findPurchaseRightQuery, userID, itemID).Scan(&id, &status, &expiresAt)
	return
}

const closePurchaseQuery = `
	UPDATE purchase_rights
	SET status = $2
	WHERE id = $1 AND status = $3 AND expires_at > now()`

func (r *PurchaseRightRepo) MarkAsUsed(ctx context.Context, purchaseID uuid.UUID) (success bool, err error) {
	tag, err := r.pool.Exec(ctx, closePurchaseQuery, purchaseID, domain.StatusUsed, domain.StatusGranted)
	if err != nil {
		return false, err
	}

	if tag.RowsAffected() == 0 {
		return false, nil
	}
	return true, nil
}
