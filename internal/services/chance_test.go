package services

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Арифметика шанса и полосы продвижения проверяется без базы: это чистые
// функции, и именно их результат пользователь видит на экране ожидания.

// TestChancePercent — P(Binomial(ahead, conv) < unitsLeft) в процентах.
// Отдельно проверяются вырожденные случаи: на них держится то, что экран
// ожидания не показывает бессмысленных чисел.
func TestChancePercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ahead     int
		unitsLeft int
		conv      float64
		want      int
	}{
		{name: "товар кончился — шанса нет", ahead: 0, unitsLeft: 0, conv: 0.5, want: 0},
		{name: "остаток ушёл в минус — всё равно ноль", ahead: 3, unitsLeft: -1, conv: 0.5, want: 0},
		{name: "впереди никого", ahead: 0, unitsLeft: 1, conv: 0.9, want: 100},
		{name: "единиц больше, чем желающих впереди", ahead: 2, unitsLeft: 3, conv: 1, want: 100},
		{name: "впереди никто не покупает — слоты вернутся", ahead: 10, unitsLeft: 1, conv: 0, want: 100},
		{name: "впереди покупают все, и их не меньше остатка", ahead: 2, unitsLeft: 2, conv: 1, want: 0},
		{name: "один впереди, монетка", ahead: 1, unitsLeft: 1, conv: 0.5, want: 50},
		{name: "трое впереди на две единицы, монетка", ahead: 3, unitsLeft: 2, conv: 0.5, want: 50},
		{name: "двое впереди на одну единицу, конверсия 0.5", ahead: 2, unitsLeft: 1, conv: 0.5, want: 25},
		{name: "длинная очередь при низкой конверсии", ahead: 10, unitsLeft: 3, conv: 0.1, want: 93},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, chancePercent(tt.ahead, tt.unitsLeft, tt.conv))
		})
	}
}

// TestChancePercent_IsMonotonic — здравый смысл в виде теста: чем длиннее
// очередь впереди, тем меньше шанс, и он никогда не выходит за 0..100.
// Пользователь, который видит, что шанс вырос от удлинившейся очереди,
// перестанет верить и цифре, и очереди.
func TestChancePercent_IsMonotonic(t *testing.T) {
	t.Parallel()

	previous := 101
	for ahead := range 40 {
		got := chancePercent(ahead, 3, 0.4)

		require.GreaterOrEqual(t, got, 0)
		require.LessOrEqual(t, got, 100)
		require.LessOrEqualf(t, got, previous,
			"шанс обязан падать с ростом очереди впереди: на %d впереди стало %d%% против %d%%", ahead, got, previous)

		previous = got
	}
}

// TestProgressPercent — полоса продвижения очереди. nil означает «рисовать не
// на чем»: пустая полоса честнее, чем полоса, посчитанная от неизвестной
// начальной позиции.
func TestProgressPercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		initial  *int
		position int
		want     *int
	}{
		{name: "начальной позиции нет — полосы нет", initial: nil, position: 5, want: nil},
		{name: "вошёл первым — двигаться некуда", initial: ptrInt(1), position: 1, want: nil},
		{name: "позиция ещё не посчитана", initial: ptrInt(9), position: 0, want: nil},
		{name: "с девятого на девятый — ноль", initial: ptrInt(9), position: 9, want: ptrInt(0)},
		{name: "с девятого на пятый", initial: ptrInt(9), position: 5, want: ptrInt(50)},
		{name: "дошёл до головы очереди", initial: ptrInt(9), position: 1, want: ptrInt(100)},
		{name: "очередь удлинилась — не уходим в минус", initial: ptrInt(4), position: 7, want: ptrInt(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := progressPercent(tt.initial, tt.position)

			if tt.want == nil {
				require.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			require.Equal(t, *tt.want, *got)
		})
	}
}

func ptrInt(value int) *int { return &value }
