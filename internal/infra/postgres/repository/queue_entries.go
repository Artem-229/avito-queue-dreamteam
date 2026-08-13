package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"avito-queue/internal/domain"
)

// QueueEntryRepository — единственный репозиторий состояния очереди и права на
// покупку. Пришёл на смену паре QueueRepository + PurchaseRightRepository:
// раньше одно участие описывалось двумя строками в разных таблицах, которые
// приходилось держать согласованными вручную.
//
// Счётчики catalog_items.granted_count/used_count двигает тот же оператор,
// который меняет статус участия: в каждом мутирующем запросе есть CTE counters
// с дельтами из counterDeltas (INV-5). Отдельного вызова «поправить счётчик»
// из Go не существует, поэтому забыть обновить одно из двух невозможно.
type QueueEntryRepository struct {
	pool *pgxpool.Pool
}

func NewQueueEntryRepo(pool *pgxpool.Pool) *QueueEntryRepository {
	return &QueueEntryRepository{pool: pool}
}

// counterDeltas — как переход статуса двигает счётчики товара. Единственное
// место в системе, где записано, какие состояния занимают единицу тиража:
// granted удерживает её, purchased расходует, остальные не касаются.

func counterDeltas(from, to domain.QueueEntryStatus) (granted, used int) {
	switch from {
	case domain.QueueEntryStatusGranted:
		granted--
	case domain.QueueEntryStatusPurchased:
		used--
	}

	switch to {
	case domain.QueueEntryStatusGranted:
		granted++
	case domain.QueueEntryStatusPurchased:
		used++
	}

	return granted, used
}

func (q *QueueEntryRepository) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
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

// Enter ставит пользователя в очередь. Идемпотентна: при активном участии
// (waiting или granted) вставки не происходит и возвращается
// ErrUserAlreadyInQueue — повторный вход не ошибка, а то же состояние.
//
// created_at не передаётся из Go: это ключ сортировки FIFO, и часы
// app-контейнера в этой роли рассинхронизировали бы порядок между репликами
// (INV-6). initial_position считается тем же запросом — позиция на входе нужна,
// чтобы показать пользователю продвижение очереди, и постфактум она невычислима.
func (q *QueueEntryRepository) Enter(ctx context.Context, userID, itemID uuid.UUID) error {
	query := `
		INSERT INTO queue_entries (user_id, item_id, status, initial_position)
		SELECT $1, $2, $3,
		       (SELECT count(*) + 1 FROM queue_entries WHERE item_id = $2 AND status = $3)
		ON CONFLICT (user_id, item_id) WHERE status IN ('waiting', 'granted') DO NOTHING`

	res, err := db(ctx, q.pool).Exec(ctx, query, userID, itemID, domain.QueueEntryStatusWaiting)
	if err != nil {
		return fmt.Errorf("inserting queue entry: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrUserAlreadyInQueue
	}

	return nil
}

// ExpireOverdue гасит участия с истёкшим сроком права и возвращает их
// пользователей. Раньше это были два запроса в двух таблицах (ExpireOld +
// MarkGrantedExpired) со сверкой числа затронутых строк: если они расходились,
// код мог только записать ошибку в лог и продолжить. Теперь строка одна и
// расходиться нечему.
//
// Сравнение с now() выполняет Postgres — та же база, что считала expires_at при
// выдаче, иначе сравнивались бы часы двух машин (INV-6).
func (q *QueueEntryRepository) ExpireOverdue(ctx context.Context, itemID uuid.UUID) ([]uuid.UUID, error) {
	if _, ok := extractTx(ctx); !ok {
		return nil, fmt.Errorf("ExpireOverdue вне транзакции: %w", domain.ErrNoTx)
	}

	grantedDelta, usedDelta := counterDeltas(domain.QueueEntryStatusGranted, domain.QueueEntryStatusExpired)

	query := `
		WITH expired AS (
			UPDATE queue_entries
			   SET status = $1, resolved_at = now()
			 WHERE item_id = $2 AND status = $3 AND expires_at < now()
			RETURNING user_id
		),
		counters AS (
			UPDATE catalog_items
			   SET granted_count = granted_count + $4 * (SELECT count(*) FROM expired),
			       used_count    = used_count    + $5 * (SELECT count(*) FROM expired)
			 WHERE id = $2 AND (SELECT count(*) FROM expired) > 0
		)
		SELECT user_id FROM expired`

	rows, err := db(ctx, q.pool).Query(ctx, query,
		domain.QueueEntryStatusExpired, itemID, domain.QueueEntryStatusGranted, grantedDelta, usedDelta)
	if err != nil {
		return nil, fmt.Errorf("expiring overdue entries: %w", err)
	}
	defer rows.Close()

	userIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scanning expired entry: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating through expired entries: %w", err)
	}

	return userIDs, nil
}

// GrantNext выдаёт право первым slots ожидающим по FIFO и возвращает их
// пользователей. Выборка и выдача — один запрос: раньше это были три шага
// (найти ожидающих, вставить право, перевести запись очереди), между которыми
// состояние могло разойтись.
//
// Блокировки в подзапросе нет намеренно: вызывающий уже держит строку товара
// через CatalogRepo.LockItem (INV-3), поэтому операции по одному товару
// сериализованы, а по разным не мешают друг другу.
//
// Срок жизни права считает база из hold_ttl_seconds товара — настройка окна на
// оплату живёт в каталоге и меняется из админки.
func (q *QueueEntryRepository) GrantNext(ctx context.Context, itemID uuid.UUID, slots int) ([]uuid.UUID, error) {
	if _, ok := extractTx(ctx); !ok {
		return nil, fmt.Errorf("GrantNext вне транзакции: %w", domain.ErrNoTx)
	}
	if slots <= 0 {
		return nil, nil
	}

	grantedDelta, usedDelta := counterDeltas(domain.QueueEntryStatusWaiting, domain.QueueEntryStatusGranted)

	query := `
		WITH next AS (
			SELECT id
			  FROM queue_entries
			 WHERE item_id = $1 AND status = $2
			 ORDER BY created_at, user_id
			 LIMIT $3
		),
		granted AS (
			UPDATE queue_entries e
			   SET status     = $4,
			       granted_at = now(),
			       expires_at = now() + make_interval(
			           secs => (SELECT hold_ttl_seconds FROM catalog_items WHERE id = $1))
			  FROM next
			 WHERE e.id = next.id
			RETURNING e.user_id
		),
		counters AS (
			UPDATE catalog_items
			   SET granted_count = granted_count + $5 * (SELECT count(*) FROM granted),
			       used_count    = used_count    + $6 * (SELECT count(*) FROM granted)
			 WHERE id = $1 AND (SELECT count(*) FROM granted) > 0
		)
		SELECT user_id FROM granted`

	rows, err := db(ctx, q.pool).Query(ctx, query,
		itemID, domain.QueueEntryStatusWaiting, slots, domain.QueueEntryStatusGranted, grantedDelta, usedDelta)
	if err != nil {
		return nil, fmt.Errorf("granting rights to waiting users: %w", err)
	}
	defer rows.Close()

	userIDs := make([]uuid.UUID, 0, slots)
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scanning granted user: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating through granted users: %w", err)
	}

	return userIDs, nil
}

// MarkPurchased тратит выданное право. Проверка срока — в самом UPDATE, чтобы
// между чтением и записью не оставалось окна: право, истёкшее миллисекундой
// раньше, не должно превратиться в покупку.
// false без ошибки — права нет, оно уже потрачено или просрочено.
func (q *QueueEntryRepository) MarkPurchased(ctx context.Context, userID, itemID uuid.UUID) (bool, error) {
	grantedDelta, usedDelta := counterDeltas(domain.QueueEntryStatusGranted, domain.QueueEntryStatusPurchased)

	query := `
		WITH bought AS (
			UPDATE queue_entries
			   SET status = $1, resolved_at = now()
			 WHERE user_id = $2 AND item_id = $3 AND status = $4 AND expires_at > now()
			RETURNING id
		),
		counters AS (
			UPDATE catalog_items
			   SET granted_count = granted_count + $5 * (SELECT count(*) FROM bought),
			       used_count    = used_count    + $6 * (SELECT count(*) FROM bought)
			 WHERE id = $3 AND (SELECT count(*) FROM bought) > 0
		)
		SELECT count(*) FROM bought`

	var affected int
	err := db(ctx, q.pool).QueryRow(ctx, query,
		domain.QueueEntryStatusPurchased, userID, itemID, domain.QueueEntryStatusGranted,
		grantedDelta, usedDelta).Scan(&affected)
	if err != nil {
		return false, fmt.Errorf("marking entry purchased: %w", err)
	}

	return affected > 0, nil
}

// UpdateStatus переводит участие из from в to. Фильтр по from обязателен: у
// пользователя может быть несколько записей по товару (старая expired, новая
// после повторного входа), и без него UPDATE задел бы обе. Заодно даёт
// CAS-семантику — ноль затронутых строк это ErrStatusConflict, а не «нечего
// делать» (INV-11).
func (q *QueueEntryRepository) UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, from, to domain.QueueEntryStatus) error {
	grantedDelta, usedDelta := counterDeltas(from, to)

	query := `
		WITH moved AS (
			UPDATE queue_entries
			   SET status = $1,
			       resolved_at = CASE WHEN $5 THEN now() ELSE resolved_at END
			 WHERE user_id = $2 AND item_id = $3 AND status = $4
			RETURNING id
		),
		counters AS (
			UPDATE catalog_items
			   SET granted_count = granted_count + $6 * (SELECT count(*) FROM moved),
			       used_count    = used_count    + $7 * (SELECT count(*) FROM moved)
			 WHERE id = $3
			   AND (SELECT count(*) FROM moved) > 0
			   AND ($6 <> 0 OR $7 <> 0)
		)
		SELECT count(*) FROM moved`

	var affected int
	err := db(ctx, q.pool).QueryRow(ctx, query,
		to, userID, itemID, from, to.IsTerminal(), grantedDelta, usedDelta).Scan(&affected)
	if err != nil {
		return fmt.Errorf("updating entry status: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("entry %s -> %s for user %s: %w", from, to, userID, domain.ErrStatusConflict)
	}

	return nil
}

// MarkSoldOut закрывает ожидание, когда весь тираж выкуплен.
//
// Условие вызова — used_count >= total_stock — проверяет Reconcile: это
// доменное правило, а не деталь хранения. Держателей права в этот момент не
// существует: инвариант granted + used <= total_stock при выкупленном тираже
// оставляет ровно ноль выданных прав, поэтому фильтр по waiting покрывает всех,
// кто ещё чего-то ждёт.
func (q *QueueEntryRepository) MarkSoldOut(ctx context.Context, itemID uuid.UUID) error {
	query := `
		UPDATE queue_entries
		   SET status = $1, resolved_at = now()
		 WHERE item_id = $2 AND status = $3`

	_, err := db(ctx, q.pool).Exec(ctx, query,
		domain.QueueEntryStatusSoldOut, itemID, domain.QueueEntryStatusWaiting)
	if err != nil {
		return fmt.Errorf("marking entries sold_out: %w", err)
	}

	return nil
}

// NeedsReconcile — дешёвая проверка без блокировки товара: есть ли для
// Reconcile работа. Нужна, чтобы поллинг статуса раз в секунду не превращал
// строку товара в горячую точку.
//
// Предикат обязан покрывать все действия Reconcile, иначе действие
// недостижимо. Помимо просроченных прав это переход в sold_out: при выкупленном
// тираже свободных слотов нет по определению, и одна лишь проверка «свободный
// слот > 0» никогда не пустила бы Reconcile выставить sold_out — ожидающий
// висел бы в waiting на распроданный товар вечно.
func (q *QueueEntryRepository) NeedsReconcile(ctx context.Context, itemID uuid.UUID) (bool, error) {
	query := `
		SELECT
			EXISTS (
				SELECT 1 FROM queue_entries
				 WHERE item_id = $1 AND status = $2 AND expires_at < now()
			)
			OR (
				EXISTS (
					SELECT 1 FROM queue_entries WHERE item_id = $1 AND status = $3
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
	err := db(ctx, q.pool).QueryRow(ctx, query, itemID,
		domain.QueueEntryStatusGranted, domain.QueueEntryStatusWaiting).Scan(&need)
	if err != nil {
		return false, fmt.Errorf("checking if reconcile is needed: %w", err)
	}

	return need, nil
}

// CountByStatus — число участий товара в заданном статусе. Reconcile считает
// granted и purchased по этой таблице, а не по счётчикам прочитанного товара:
// строку товара вернул LockItem в начале транзакции, а ExpireOverdue успел
// сдвинуть счётчики после этого, и в структуре в памяти они уже устарели.
func (q *QueueEntryRepository) CountByStatus(ctx context.Context, itemID uuid.UUID, status domain.QueueEntryStatus) (int, error) {
	query := `SELECT count(*) FROM queue_entries WHERE item_id = $1 AND status = $2`

	var count int
	if err := db(ctx, q.pool).QueryRow(ctx, query, itemID, status).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting entries in status %s: %w", status, err)
	}

	return count, nil
}

// Outcomes — судьба выданных прав по товару: сколько дошло до покупки и сколько
// сгорело. Это знаменатель конверсии; права, которые ещё держат, в него не
// входят — их исход неизвестен.
func (q *QueueEntryRepository) Outcomes(ctx context.Context, itemID uuid.UUID) (purchased, expired int, err error) {
	query := `
		SELECT count(*) FILTER (WHERE status = $2),
		       count(*) FILTER (WHERE status = $3)
		  FROM queue_entries
		 WHERE item_id = $1`

	err = db(ctx, q.pool).QueryRow(ctx, query, itemID,
		domain.QueueEntryStatusPurchased, domain.QueueEntryStatusExpired).Scan(&purchased, &expired)
	if err != nil {
		return 0, 0, fmt.Errorf("counting item outcomes: %w", err)
	}

	return purchased, expired, nil
}

// OutcomesTotal — то же по всей системе. Нужен, пока по товару не набралось
// статистики: конверсия по двум завершённым правам не значит ничего.
func (q *QueueEntryRepository) OutcomesTotal(ctx context.Context) (purchased, expired int, err error) {
	query := `
		SELECT count(*) FILTER (WHERE status = $1),
		       count(*) FILTER (WHERE status = $2)
		  FROM queue_entries`

	err = db(ctx, q.pool).QueryRow(ctx, query,
		domain.QueueEntryStatusPurchased, domain.QueueEntryStatusExpired).Scan(&purchased, &expired)
	if err != nil {
		return 0, 0, fmt.Errorf("counting total outcomes: %w", err)
	}

	return purchased, expired, nil
}

// GetEntry — последнее участие пользователя в очереди на товар. Записи не
// удаляются, поэтому после повторного входа их несколько, и актуальна самая
// свежая.
func (q *QueueEntryRepository) GetEntry(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueEntry, error) {
	query := `
		SELECT id, user_id, item_id, status, created_at, granted_at, expires_at, resolved_at, initial_position
		  FROM queue_entries
		 WHERE user_id = $1 AND item_id = $2
		 ORDER BY created_at DESC
		 LIMIT 1`

	var entry domain.QueueEntry
	err := db(ctx, q.pool).QueryRow(ctx, query, userID, itemID).Scan(
		&entry.ID,
		&entry.UserID,
		&entry.ItemID,
		&entry.Status,
		&entry.CreatedAt,
		&entry.GrantedAt,
		&entry.ExpiresAt,
		&entry.ResolvedAt,
		&entry.InitialPosition,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.QueueEntry{}, domain.ErrUserNotFound
		}
		return domain.QueueEntry{}, fmt.Errorf("getting queue entry: %w", err)
	}

	return entry, nil
}

// Snapshot — всё, что нужно конверту статуса: участие пользователя, его
// позиция, размер очереди и время сервера. Два запроса вместо четырёх, которые
// делал сервис раньше; эту ручку фронт опрашивает раз в секунду, и каждое
// обращение умножается на число ожидающих.
//
// Отсутствие участия — не ошибка, а состояние not_in_queue (INV-8): Entry
// остаётся пустым.
func (q *QueueEntryRepository) Snapshot(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueSnapshot, error) {
	var snapshot domain.QueueSnapshot

	entry, err := q.GetEntry(ctx, userID, itemID)
	switch {
	case err == nil:
		snapshot.Entry = &entry
	case !errors.Is(err, domain.ErrUserNotFound):
		return domain.QueueSnapshot{}, fmt.Errorf("getting entry for snapshot: %w", err)
	}

	// Позиция считается по тому же ключу (created_at, user_id), по которому
	// сортирует GrantNext: при совпадающих created_at показанная позиция иначе
	// разошлась бы с реальным порядком выдачи (INV-10).
	//
	// Когда участия нет или оно не в статусе waiting, ключ приходит пустым,
	// сравнение даёт NULL и в счётчик ничего не попадает. Позиция в этом случае
	// бессмысленна, и ниже она не читается.
	const query = `
		SELECT now(),
		       counters.queue_size,
		       counters.position,
		       i.total_stock, i.granted_count, i.used_count
		  FROM catalog_items i
		  CROSS JOIN LATERAL (
			SELECT count(*) FILTER (WHERE status = $2) AS queue_size,
			       count(*) FILTER (WHERE status = $2 AND (created_at, user_id) < ($3, $4)) + 1 AS position
			  FROM queue_entries
			 WHERE item_id = i.id
		  ) counters
		 WHERE i.id = $1 AND i.deleted_at IS NULL`

	var createdAt *time.Time
	var waitingUserID *uuid.UUID
	if snapshot.Entry != nil && snapshot.Entry.Status == domain.QueueEntryStatusWaiting {
		createdAt = &snapshot.Entry.CreatedAt
		waitingUserID = &userID
	}

	var position int
	err = db(ctx, q.pool).QueryRow(ctx, query, itemID, domain.QueueEntryStatusWaiting, createdAt, waitingUserID).
		Scan(&snapshot.ServerTime, &snapshot.QueueSize, &position,
			&snapshot.TotalStock, &snapshot.GrantedCount, &snapshot.UsedCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.QueueSnapshot{}, domain.ErrNoItemFound
		}
		return domain.QueueSnapshot{}, fmt.Errorf("getting queue counters: %w", err)
	}

	if waitingUserID != nil {
		snapshot.Position = position
	}

	return snapshot, nil
}

// ExpireNow — демо-ручка: двигает срок всех выданных прав на товар в прошлое,
// дальше их находит обычный ExpireOverdue. Нужна, чтобы демонстрация истечения
// права не занимала hold_ttl_seconds реального ожидания.
func (q *QueueEntryRepository) ExpireNow(ctx context.Context, itemID uuid.UUID) error {
	query := `
		UPDATE queue_entries
		   SET expires_at = now() - interval '1 second'
		 WHERE item_id = $1 AND status = $2`

	_, err := db(ctx, q.pool).Exec(ctx, query, itemID, domain.QueueEntryStatusGranted)
	if err != nil {
		return fmt.Errorf("expiring rights now: %w", err)
	}

	return nil
}
