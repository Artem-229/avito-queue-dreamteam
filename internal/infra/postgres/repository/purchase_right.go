package repository

import (
	"avito-queue/internal/domain"
	"context"
	"errors"
	"fmt"

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
		INSERT INTO purchase_rights (item_id, user_id, status, expires_at)
		SELECT $1, $2, $3, now() + make_interval(secs => i.hold_ttl_seconds)
		  FROM catalog_items i
		 WHERE i.id = $1
		ON CONFLICT (user_id, item_id) WHERE status = 'granted' DO NOTHING`

	res, err := db(ctx, r.pool).Exec(ctx, query, itemID, userID, domain.PurchaseRightStatusGranted)
	if err != nil {
		return fmt.Errorf("creating purchase_right: %w", err)
	}
	// Ноль строк — товар исчез или у пользователя уже есть активное право
	// (сработал ON CONFLICT); в обоих случаях это рассогласование состояния,
	// а не сбой БД, поэтому нужен сентинел, а не безликий 500.
	if res.RowsAffected() == 0 {
		return fmt.Errorf("granting right to user %s on item %s: %w", userID, itemID, domain.ErrStatusConflict)
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
	err := db(ctx, r.pool).QueryRow(ctx, query, userID, itemID).Scan(&right.ID, &right.Status, &right.UserID, &right.ItemID, &right.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PurchaseRight{}, domain.ErrNoPurchaseRight
		}
		return domain.PurchaseRight{}, fmt.Errorf("getting purchase_right: %w", err)
	}

	return right, nil
}

// ExpireOld гасит права, у которых вышел срок. Сравнение с now() идёт внутри
// Postgres, той же базы, что считала expires_at в Create — иначе сравнивались
// бы часы двух разных машин (INV-6).
func (r *PurchaseRightRepository) ExpireOld(ctx context.Context, itemID uuid.UUID) ([]uuid.UUID, error) {
	query := `
			UPDATE purchase_rights
			SET status = $1 WHERE item_id = $2
			AND status = $3
			AND expires_at < now()
			RETURNING user_id`

	var IDs []uuid.UUID
	rows, err := db(ctx, r.pool).Query(ctx, query, domain.PurchaseRightStatusExpired, itemID, domain.PurchaseRightStatusGranted)
	if err != nil {
		return nil, fmt.Errorf("expiring purchase rights: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ID uuid.UUID
		if err := rows.Scan(&ID); err != nil {
			return nil, fmt.Errorf("scanning purchase_right: %w", err)
		}
		IDs = append(IDs, ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating through expired purchase rights: %w", err)
	}

	return IDs, nil
}

// UpdateStatus переводит право из from в to. Фильтр по from обязателен: у
// пользователя может быть несколько прав по товару (истёкшее плюс новое), и
// UPDATE без фильтра переписал бы историю обоих разом.
func (r *PurchaseRightRepository) UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, from, to domain.PurchaseRightStatus) error {
	query := `UPDATE purchase_rights SET status = $1 WHERE user_id = $2 AND item_id = $3 AND status = $4`

	tag, err := db(ctx, r.pool).Exec(ctx, query, to, userID, itemID, from)
	if err != nil {
		return fmt.Errorf("update purchase_right status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("purchase right %s -> %s for user %s: %w", from, to, userID, domain.ErrStatusConflict)
	}

	return nil
}

// ExpireNow — демо-ручка: двигает expires_at всех выданных прав на товар в
// прошлое, дальше их находит обычный ExpireOld.
func (r *PurchaseRightRepository) ExpireNow(ctx context.Context, itemID uuid.UUID) error {
	query := `
		UPDATE purchase_rights
		   SET expires_at = now() - interval '1 second'
		 WHERE item_id = $1 AND status = $2`

	_, err := db(ctx, r.pool).Exec(ctx, query, itemID, domain.PurchaseRightStatusGranted)
	if err != nil {
		return fmt.Errorf("expiring purchase rights now: %w", err)
	}
	return nil
}

func (r *PurchaseRightRepository) CountGranted(ctx context.Context, itemID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM purchase_rights WHERE item_id = $1 AND status = $2`

	var count int
	err := db(ctx, r.pool).QueryRow(ctx, query, itemID, domain.PurchaseRightStatusGranted).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting purchase_right %w", err)
	}

	return count, nil
}

func (r *PurchaseRightRepository) CountUsed(ctx context.Context, itemID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM purchase_rights WHERE item_id = $1 AND status = $2`

	var count int
	err := db(ctx, r.pool).QueryRow(ctx, query, itemID, domain.PurchaseRightStatusUsed).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting purchase_right %w", err)
	}

	return count, nil
}

func (r *PurchaseRightRepository) MarkAsUsed(ctx context.Context, userID, itemID uuid.UUID) (success bool, err error) {
	query := `
		UPDATE purchase_rights SET status = $3
		WHERE user_id = $1 AND item_id = $2 AND status = $4 AND expires_at > now()`

	tag, err := db(ctx, r.pool).Exec(ctx, query, userID, itemID, domain.PurchaseRightStatusUsed, domain.PurchaseRightStatusGranted)
	if err != nil {
		return false, err
	}

	if tag.RowsAffected() == 0 {
		return false, nil
	}
	return true, nil
}
