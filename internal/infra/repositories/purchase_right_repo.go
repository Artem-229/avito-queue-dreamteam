package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PurchaseRightRepo struct {
	pool *pgxpool.Pool
}

func NewPurchaseRightRepo(pool *pgxpool.Pool) *PurchaseRightRepo {
	return &PurchaseRightRepo{pool: pool}
}

const findPurchaseRightQuery = `
	SELECT id_purchase, purchase_status, expires_at
	FROM purchase_rights
	WHERE id_user = $1 AND id_item = $2`

func (r *PurchaseRightRepo) FindByUserAndItem(ctx context.Context, userID, itemID int) (id int, status string, expiresAt time.Time, err error) {
	err = r.pool.QueryRow(ctx, findPurchaseRightQuery, userID, itemID).Scan(&id, &status, &expiresAt)
	return
}
