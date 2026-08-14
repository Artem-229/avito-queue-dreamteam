package services

import (
	"context"
	"fmt"

	"avito-queue/internal/domain"
)

type StatsRepository interface {
	Collect(ctx context.Context) (domain.QueueStats, error)
}

// overdueExpirer — минимальный контракт, который нужен GetStats, чтобы
// метрики не зависели от того, заходил ли кто-то на конкретную карточку
// товара. Определён здесь, а не в queue_entries.go: QueueEntryRepository
// уже удовлетворяет этому интерфейсу структурно (метод ExpireAllOverdue),
// отдельно менять интерфейс queueEntriesRepository не нужно.
type overdueExpirer interface {
	ExpireAllOverdue(ctx context.Context) error
}

type StatsService struct {
	repo    StatsRepository
	expirer overdueExpirer
}

func NewStatsService(repo StatsRepository, expirer overdueExpirer) *StatsService {
	return &StatsService{repo: repo, expirer: expirer}
}

// GetStats отдаёт сводку по очереди. Перед чтением сначала гасит все
// просроченные granted-права одним batch-запросом (ExpireAllOverdue) —
// иначе цифры были бы актуальны только для товаров, которые кто-то недавно
// открывал, а не для каталога в целом. Конверсия считается здесь, а не в
// SQL: это правило продукта («доля прав, дошедших до покупки»), и ему
// место рядом с остальной доменной арифметикой, где его видно и можно
// проверить тестом.
func (s *StatsService) GetStats(ctx context.Context) (domain.QueueStats, error) {
	if err := s.expirer.ExpireAllOverdue(ctx); err != nil {
		return domain.QueueStats{}, fmt.Errorf("services.GetStats (expiring overdue): %w", err)
	}

	stats, err := s.repo.Collect(ctx)
	if err != nil {
		return domain.QueueStats{}, fmt.Errorf("services.GetStats: %w", err)
	}

	stats.Conversion = conversion(stats.Purchased, stats.Expired)

	return stats, nil
}

// conversion — доля покупок среди завершившихся прав. nil вместо нуля, когда
// завершившихся прав нет: ноль означал бы «никто не покупает», а правда здесь
// «ещё нечего мерить», и на дашборде это разные вещи.
func conversion(purchased, expired int) *float64 {
	finished := purchased + expired
	if finished <= 0 {
		return nil
	}

	value := float64(purchased) / float64(finished)

	return &value
}
