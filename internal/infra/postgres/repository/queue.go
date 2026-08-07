package repository

import (
	"avito-queue/internal/domain"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionKey string

const key TransactionKey = "transaction_key"

type QueueRepository struct {
	pool *pgxpool.Pool
}

func NewQueueRepo(pool *pgxpool.Pool) *QueueRepository {
	return &QueueRepository{pool: pool}
}

func (q *QueueRepository) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := q.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txCtx := context.WithValue(ctx, key, tx)

	if err := fn(txCtx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func ExtractTx(ctx context.Context) (pgx.Tx, bool) {
	if tx, ok := ctx.Value(key).(pgx.Tx); ok {
		return tx, true
	}
	return nil, false
}

func (q *QueueRepository) Entry(ctx context.Context, userID, itemID uuid.UUID) error {
	query := `
		INSERT INTO queue (user_id, item_id, status, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, item_id) WHERE status IN ('waiting', 'granted') DO NOTHING;`

	res, err := q.pool.Exec(ctx, query, userID, itemID, domain.QueueStatusWaiting, time.Now())
	if err != nil {
		return fmt.Errorf("inserting purchase rights: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrUserAlreadyInQueue
	}

	return nil
}

func (q *QueueRepository) MarkRecordsExpired(ctx context.Context, itemID uuid.UUID, userIDs []uuid.UUID) error {
	query := `
		UPDATE queue 
		SET status = $1 
		WHERE item_id = $2 
		  AND user_id = ANY($3)
		  AND status = $4`

	tx, ok := ExtractTx(ctx)
	if !ok {
		return fmt.Errorf("could not extract tx")
	}

	_, err := tx.Exec(ctx, query, domain.QueueStatusExpired, itemID, userIDs, domain.QueueStatusWaiting)
	if err != nil {
		return fmt.Errorf("marking queue records expired: %w", err)
	}

	return nil
}

func (q *QueueRepository) GetWaiting(ctx context.Context, itemID uuid.UUID, freeSlots int) ([]uuid.UUID, error) {
	query := `
			SELECT user_id FROM queue
			WHERE item_id = $1 AND status = $2
			ORDER BY created_at ASC
			LIMIT $3`

	rows, err := q.pool.Query(ctx, query, itemID, domain.QueueStatusWaiting, freeSlots)
	if err != nil {
		return nil, fmt.Errorf("getting waiting users from queue: %w", err)
	}

	defer rows.Close()

	userIDs := make([]uuid.UUID, 0, freeSlots)
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scanning user from queue: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating thought users ids from queue: %w", err)
	}

	return userIDs, nil
}

func (r *QueueRepository) UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, status domain.QueueStatus) error {
	query := `UPDATE queue SET status = $1 WHERE user_id = $2 AND item_id = $3`
	_, err := r.pool.Exec(ctx, query, status, userID, itemID)
	if err != nil {
		return fmt.Errorf("update purchase_right status: %w", err)
	}
	return nil
}
