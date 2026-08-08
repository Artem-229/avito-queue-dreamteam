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

func (q *QueueRepository) UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, status domain.QueueStatus) error {
	query := `UPDATE queue SET status = $1 WHERE user_id = $2 AND item_id = $3`
	_, err := q.pool.Exec(ctx, query, status, userID, itemID)
	if err != nil {
		return fmt.Errorf("update purchase_right status: %w", err)
	}
	return nil
}

func (q *QueueRepository) MarkSoldOut(ctx context.Context, itemID uuid.UUID) error {
	query := `UPDATE queue SET status = $1 WHERE item_id = $2 and status = $3`

	_, err := q.pool.Exec(ctx, query, domain.QueueStatusSoldOut, itemID, domain.QueueStatusWaiting)
	if err != nil {
		return fmt.Errorf("marking sold_out: %w", err)
	}

	return nil
}

func (q *QueueRepository) GetRecord(ctx context.Context, userID, itemID uuid.UUID) (domain.Queue, error) {
	query := `
		SELECT id, user_id, item_id, status, created_at, deleted_at
		FROM queue
		WHERE user_id = $1 AND item_id = $2
		ORDER BY created_at DESC
		LIMIT 1`

	var queueRecord domain.Queue
	err := q.pool.QueryRow(ctx, query, userID, itemID).Scan(
		&queueRecord.ID,
		&queueRecord.UserID,
		&queueRecord.ItemID,
		&queueRecord.Status,
		&queueRecord.CreatedAt,
		&queueRecord.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Queue{}, domain.ErrUserNotFound
		}
		return domain.Queue{}, fmt.Errorf("getting queue record: %w", err)
	}

	return queueRecord, nil
}

// GetPosition возвращает позицию пользователя в очереди (1 - следующий на выдачу
// права): считает, сколько записей в статусе waiting встали в очередь раньше него.
func (q *QueueRepository) GetPosition(ctx context.Context, queueRecord domain.Queue) (int, error) {
	query := `SELECT COUNT(*) FROM queue WHERE item_id = $1 AND status = $2 AND created_at < $3`

	var ahead int
	err := q.pool.QueryRow(ctx, query, queueRecord.ItemID, domain.QueueStatusWaiting, queueRecord.CreatedAt).Scan(&ahead)
	if err != nil {
		return 0, fmt.Errorf("counting queue position: %w", err)
	}

	return ahead + 1, nil
}
