package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"avito-queue/internal/domain"
)

// StatsRepository — только чтение и только агрегаты. Отдельный репозиторий, а
// не метод в существующих: запросы здесь ходят сразу по трём таблицам и не
// принадлежат ни одной из них.
type StatsRepository struct {
	pool *pgxpool.Pool
}

func NewStatsRepository(pool *pgxpool.Pool) *StatsRepository {
	return &StatsRepository{pool: pool}
}

// itemStatsQuery считает очередь в разрезе товаров. LEFT JOIN, чтобы товары
// без единого участника очереди тоже попали в таблицу с нулями: строка,
// исчезающая при отсутствии активности, читается как «товар пропал».
const itemStatsQuery = `
SELECT c.id,
       c.name,
       COUNT(*) FILTER (WHERE e.status = 'waiting') AS waiting,
       COUNT(*) FILTER (WHERE e.status = 'granted') AS granted,
       GREATEST(c.total_stock - c.granted_count - c.used_count, 0) AS stock_left
  FROM catalog_items c
  LEFT JOIN queue_entries e ON e.item_id = c.id
 WHERE c.deleted_at IS NULL
 GROUP BY c.id, c.name, c.total_stock, c.granted_count, c.used_count
 ORDER BY waiting DESC, granted DESC, c.name`

// rightsStatsQuery — судьба выданных прав. Считается по тем же статусам
// участия: право — то, что мы обещали пользователю, и честность демонстрации
// меряется именно по нему.
const rightsStatsQuery = `
SELECT COUNT(*) FILTER (WHERE status = 'purchased') AS purchased,
       COUNT(*) FILTER (WHERE status = 'expired')   AS expired
  FROM queue_entries`

// avgWaitQuery — среднее ожидание от постановки в очередь до выдачи права.
// После слияния таблиц это разность двух колонок одной строки: раньше здесь
// был LATERAL, подбиравший запись очереди к каждому праву, потому что связать
// их напрямую было нечем. AVG по пустому множеству даёт NULL — отсюда
// указатель в Go.
const avgWaitQuery = `
SELECT AVG(EXTRACT(EPOCH FROM (granted_at - created_at)))
  FROM queue_entries
 WHERE granted_at IS NOT NULL`

// Collect собирает сводку тремя запросами без транзакции: это витрина для
// дашборда, а не источник решения о выдаче права. Транзакция дала бы
// согласованный снимок ценой блокировок на пути у покупателей — цена выше
// пользы, поллинг всё равно обновит цифры через пару секунд.
func (r *StatsRepository) Collect(ctx context.Context) (domain.QueueStats, error) {
	stats := domain.QueueStats{Items: []domain.ItemStats{}}

	rows, err := db(ctx, r.pool).Query(ctx, itemStatsQuery)
	if err != nil {
		return domain.QueueStats{}, fmt.Errorf("querying item stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.ItemStats
		if err := rows.Scan(
			&item.ItemID,
			&item.Title,
			&item.Waiting,
			&item.Granted,
			&item.StockLeft,
		); err != nil {
			return domain.QueueStats{}, fmt.Errorf("scanning item stats: %w", err)
		}

		stats.TotalWaiting += item.Waiting
		stats.TotalGranted += item.Granted
		stats.Items = append(stats.Items, item)
	}

	if err := rows.Err(); err != nil {
		return domain.QueueStats{}, fmt.Errorf("rows error in item stats: %w", err)
	}

	if err := db(ctx, r.pool).QueryRow(ctx, rightsStatsQuery).Scan(
		&stats.Purchased,
		&stats.Expired,
	); err != nil {
		return domain.QueueStats{}, fmt.Errorf("querying rights stats: %w", err)
	}

	var avgWait *float64
	if err := db(ctx, r.pool).QueryRow(ctx, avgWaitQuery).Scan(&avgWait); err != nil {
		return domain.QueueStats{}, fmt.Errorf("querying average wait: %w", err)
	}

	if avgWait != nil {
		seconds := int(*avgWait)
		stats.AvgWaitSeconds = &seconds
	}

	return stats, nil
}
