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

type transactionKey string

const key transactionKey = "transaction_key"

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

// Entry ставит пользователя в очередь. created_at не передаётся из Go —
// колонка заполняется DEFAULT now() самой базой, потому что created_at ключ
// сортировки FIFO, и часы app-контейнера в этой роли рассинхронизировали бы
// порядок между репликами (INV-6).
func (q *QueueRepository) Entry(ctx context.Context, userID, itemID uuid.UUID) error {
	query := `
		INSERT INTO queue (user_id, item_id, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, item_id) WHERE status IN ('waiting', 'granted') DO NOTHING;`

	res, err := db(ctx, q.pool).Exec(ctx, query, userID, itemID, domain.QueueStatusWaiting)
	if err != nil {
		return fmt.Errorf("inserting queue record: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrUserAlreadyInQueue
	}

	return nil
}

// MarkGrantedExpired переводит в expired записи очереди тех, у кого сгорело
// уже выданное право — их queue.status на этот момент 'granted', не 'waiting'.
// Возвращает число затронутых строк, чтобы вызывающий мог сверить его с
// количеством userIDs и заметить рассинхронизацию purchase_rights/queue (INV-5).
func (q *QueueRepository) MarkGrantedExpired(ctx context.Context, itemID uuid.UUID, userIDs []uuid.UUID) (int64, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}

	query := `
		UPDATE queue
		SET status = $1
		WHERE item_id = $2
		  AND user_id = ANY($3)
		  AND status = $4`

	tag, err := db(ctx, q.pool).Exec(ctx, query, domain.QueueStatusExpired, itemID, userIDs, domain.QueueStatusGranted)
	if err != nil {
		return 0, fmt.Errorf("marking queue records expired: %w", err)
	}

	return tag.RowsAffected(), nil
}

// NeedsReconcile — дешёвая проверка без блокировки товара, чтобы частый
// поллинг статуса не превращал её в горячую точку.
//
// Предикат обязан покрывать все действия Reconcile, иначе действие
// недостижимо: помимо просроченных прав это ещё и переход в sold_out — при
// выкупленном стоке свободных слотов нет по определению, и одна лишь проверка
// "свободный слот > 0" никогда не пустила бы Reconcile выставить sold_out
// (тогда ожидающий висел бы в waiting на распроданный товар вечно, A-09).
func (q *QueueRepository) NeedsReconcile(ctx context.Context, itemID uuid.UUID) (bool, error) {
	query := `
		SELECT
			EXISTS (
				SELECT 1 FROM purchase_rights
				 WHERE item_id = $1 AND status = $2 AND expires_at < now()
			)
			OR (
				EXISTS (
					SELECT 1 FROM queue WHERE item_id = $1 AND status = $3
				)
				AND EXISTS (
					SELECT 1 FROM catalog_items
					 WHERE id = $1 AND deleted_at IS NULL
					   AND (
					        total_stock - granted_count - used_count > 0
					        OR used_count >= total_stock
					   )
				)
			)`

	var need bool
	err := db(ctx, q.pool).QueryRow(ctx, query, itemID, domain.PurchaseRightStatusGranted, domain.QueueStatusWaiting).Scan(&need)
	if err != nil {
		return false, fmt.Errorf("checking if reconcile is needed: %w", err)
	}

	return need, nil
}

func (q *QueueRepository) GetWaiting(ctx context.Context, itemID uuid.UUID, freeSlots int) ([]uuid.UUID, error) {
	query := `
			SELECT user_id FROM queue
			WHERE item_id = $1 AND status = $2
			ORDER BY created_at, user_id ASC
			LIMIT $3`

	rows, err := db(ctx, q.pool).Query(ctx, query, itemID, domain.QueueStatusWaiting, freeSlots)
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

// CountWaiting — размер очереди для конверта статуса (queue_size).
func (q *QueueRepository) CountWaiting(ctx context.Context, itemID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM queue WHERE item_id = $1 AND status = $2`

	var count int
	err := db(ctx, q.pool).QueryRow(ctx, query, itemID, domain.QueueStatusWaiting).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting waiting queue records: %w", err)
	}

	return count, nil
}

// Now — время Postgres для server_time в конверте статуса (INV-6).
func (q *QueueRepository) Now(ctx context.Context) (time.Time, error) {
	var t time.Time
	if err := db(ctx, q.pool).QueryRow(ctx, `SELECT now()`).Scan(&t); err != nil {
		return time.Time{}, fmt.Errorf("getting server time: %w", err)
	}

	return t, nil
}

// UpdateStatus переводит запись очереди из from в to. Фильтр по from
// обязателен: у пользователя может быть несколько записей по товару (старая
// expired, новая после повторного входа), и без фильтра UPDATE задел бы обе.
// Заодно даёт CAS-семантику: 0 затронутых строк — ошибка, а не «нечего делать».
func (q *QueueRepository) UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, from, to domain.QueueStatus) error {
	query := `UPDATE queue SET status = $1 WHERE user_id = $2 AND item_id = $3 AND status = $4`

	tag, err := db(ctx, q.pool).Exec(ctx, query, to, userID, itemID, from)
	if err != nil {
		return fmt.Errorf("update queue status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("queue %s -> %s for user %s: %w", from, to, userID, domain.ErrStatusConflict)
	}

	return nil
}

func (q *QueueRepository) MarkSoldOut(ctx context.Context, itemID uuid.UUID) error {
	query := `UPDATE queue SET status = $1 WHERE item_id = $2 and status = $3`

	_, err := db(ctx, q.pool).Exec(ctx, query, domain.QueueStatusSoldOut, itemID, domain.QueueStatusWaiting)
	if err != nil {
		return fmt.Errorf("marking sold_out: %w", err)
	}

	return nil
}

func (q *QueueRepository) GetRecord(ctx context.Context, userID, itemID uuid.UUID) (domain.Queue, error) {
	query := `
		SELECT id, user_id, item_id, status, created_at
		FROM queue
		WHERE user_id = $1 AND item_id = $2
		ORDER BY created_at DESC
		LIMIT 1`

	var queueRecord domain.Queue
	err := db(ctx, q.pool).QueryRow(ctx, query, userID, itemID).Scan(
		&queueRecord.ID,
		&queueRecord.UserID,
		&queueRecord.ItemID,
		&queueRecord.Status,
		&queueRecord.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Queue{}, domain.ErrUserNotFound
		}
		return domain.Queue{}, fmt.Errorf("getting queue record: %w", err)
	}

	return queueRecord, nil
}

// GetPosition — позиция в очереди (1 = следующий на выдачу права). Сравнение
// идёт по кортежу (created_at, user_id), тому же ключу, по которому сортирует
// GetWaiting — иначе при равных created_at позиция разойдётся с реальным
// порядком выдачи (INV-10).
func (q *QueueRepository) GetPosition(ctx context.Context, queueRecord domain.Queue) (int, error) {
	query := `
			SELECT COUNT(*)
			FROM queue
			WHERE item_id = $1 AND status = $2 AND (created_at, user_id) < ($3, $4)`

	var ahead int
	err := db(ctx, q.pool).QueryRow(ctx, query,
		queueRecord.ItemID, domain.QueueStatusWaiting, queueRecord.CreatedAt, queueRecord.UserID,
	).Scan(&ahead)
	if err != nil {
		return 0, fmt.Errorf("counting queue position: %w", err)
	}

	return ahead + 1, nil
}
