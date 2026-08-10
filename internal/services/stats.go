package services

import (
	"context"
	"fmt"

	"avito-queue/internal/domain"
)

type StatsRepository interface {
	Collect(ctx context.Context) (domain.QueueStats, error)
}

type StatsService struct {
	repo StatsRepository
}

func NewStatsService(repo StatsRepository) *StatsService {
	return &StatsService{repo: repo}
}

// GetStats отдаёт сводку по очереди. Конверсия считается здесь, а не в SQL:
// это правило продукта («доля прав, дошедших до покупки»), и ему место рядом с
// остальной доменной арифметикой, где его видно и можно проверить тестом.
func (s *StatsService) GetStats(ctx context.Context) (domain.QueueStats, error) {
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
