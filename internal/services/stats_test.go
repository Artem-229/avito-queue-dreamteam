package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"avito-queue/internal/domain"
)

// mockStatsRepo отдаёт заранее заданную сводку: конверсию считает сервис, и
// проверяем здесь именно её, а не SQL.
type mockStatsRepo struct {
	stats domain.QueueStats
	err   error
}

func (m *mockStatsRepo) Collect(_ context.Context) (domain.QueueStats, error) {
	return m.stats, m.err
}

func TestGetStatsConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		purchased int
		expired   int
		want      *float64
	}{
		{name: "нет завершённых прав — мерить нечего", purchased: 0, expired: 0, want: nil},
		{name: "все права дошли до покупки", purchased: 4, expired: 0, want: ptr(1.0)},
		{name: "все права сгорели", purchased: 0, expired: 3, want: ptr(0.0)},
		{name: "три из четырёх", purchased: 3, expired: 1, want: ptr(0.75)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewStatsService(&mockStatsRepo{stats: domain.QueueStats{
				Purchased: tt.purchased,
				Expired:   tt.expired,
			}})

			stats, err := service.GetStats(context.Background())
			require.NoError(t, err)

			if tt.want == nil {
				require.Nil(t, stats.Conversion)
				return
			}

			require.NotNil(t, stats.Conversion)
			require.InDelta(t, *tt.want, *stats.Conversion, 1e-9)
		})
	}
}

// Удерживаемые права в знаменатель не входят: пока право живо, его судьба
// неизвестна, и включение занижало бы конверсию тем сильнее, чем длиннее
// очередь прямо сейчас.
func TestGetStatsConversionIgnoresActiveRights(t *testing.T) {
	t.Parallel()

	service := NewStatsService(&mockStatsRepo{stats: domain.QueueStats{
		Purchased:    1,
		Expired:      1,
		TotalGranted: 98,
	}})

	stats, err := service.GetStats(context.Background())
	require.NoError(t, err)
	require.NotNil(t, stats.Conversion)
	require.InDelta(t, 0.5, *stats.Conversion, 1e-9)
}

func TestGetStatsPropagatesRepoError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("collect failed")
	service := NewStatsService(&mockStatsRepo{err: sentinel})

	_, err := service.GetStats(context.Background())
	require.ErrorIs(t, err, sentinel)
}

func ptr(value float64) *float64 { return &value }
