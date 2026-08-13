package services

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Шанс на покупку — честный ответ на вопрос «дойдёт ли до меня очередь».

const (
	// conversionTTL — как долго живёт посчитанная конверсия. Конверт статуса
	// опрашивается раз в секунду каждым ожидающим: агрегат на каждый опрос
	// превратил бы честную оценку в источник нагрузки.
	conversionTTL = 30 * time.Second
	// conversionMinSample — сколько завершённых прав нужно, чтобы верить
	// статистике товара. Меньше — берём общую по системе, иначе одна случайная
	// покупка дала бы конверсию 100%.
	conversionMinSample = 5
	// conversionDefault — пока не завершилось ни одного права. Ровно
	// «неизвестно», без оптимизма и без пессимизма.
	conversionDefault = 0.5

	conversionBasisItem    = "item"
	conversionBasisGlobal  = "global"
	conversionBasisDefault = "default"
)

// outcomesRepo — исходы выданных прав. В знаменатель конверсии входят только
// завершённые: те, что ещё держат, ничего не говорят о будущем, а их учёт
// занижал бы конверсию тем сильнее, чем длиннее очередь прямо сейчас.
type outcomesRepo interface {
	Outcomes(ctx context.Context, itemID uuid.UUID) (purchased, expired int, err error)
	OutcomesTotal(ctx context.Context) (purchased, expired int, err error)
}

type conversionRate struct {
	value     float64
	basis     string
	expiresAt time.Time
}

// ChanceCalculator считает шанс и кэширует конверсию в памяти.
// Кэш процессный: цифра справочная, расхождение между репликами на десяток
// секунд ни на что не влияет, а поход в общее хранилище за ней стоил бы
// дороже самой цифры.
type ChanceCalculator struct {
	repo outcomesRepo

	mu     sync.Mutex
	byItem map[uuid.UUID]conversionRate
	global conversionRate
}

func NewChanceCalculator(repo outcomesRepo) *ChanceCalculator {
	return &ChanceCalculator{
		repo:   repo,
		byItem: make(map[uuid.UUID]conversionRate),
	}
}

// For — шанс для того, кто стоит на позиции position (1 = следующий на выдачу).
// Для оценки до входа в очередь передаётся позиция queueSize+1: человек встал
// бы в хвост.
func (c *ChanceCalculator) For(ctx context.Context, itemID uuid.UUID, position, grantedCount, usedCount, totalStock int) (int, string, error) {
	conv, err := c.conversionFor(ctx, itemID)
	if err != nil {
		return 0, "", fmt.Errorf("getting conversion: %w", err)
	}

	// Впереди — держатели прав и все, кто раньше в очереди.
	ahead := grantedCount + position - 1
	unitsLeft := totalStock - usedCount

	return chancePercent(ahead, unitsLeft, conv.value), conv.basis, nil
}

func (c *ChanceCalculator) conversionFor(ctx context.Context, itemID uuid.UUID) (conversionRate, error) {
	c.mu.Lock()
	cached, ok := c.byItem[itemID]
	c.mu.Unlock()

	if ok && time.Now().Before(cached.expiresAt) {
		return cached, nil
	}

	purchased, expired, err := c.repo.Outcomes(ctx, itemID)
	if err != nil {
		return conversionRate{}, fmt.Errorf("counting item outcomes: %w", err)
	}

	conv := conversionRate{expiresAt: time.Now().Add(conversionTTL)}
	if resolved := purchased + expired; resolved >= conversionMinSample {
		conv.value = float64(purchased) / float64(resolved)
		conv.basis = conversionBasisItem
	} else {
		global, err := c.globalConversion(ctx)
		if err != nil {
			return conversionRate{}, err
		}
		conv.value, conv.basis = global.value, global.basis
	}

	c.mu.Lock()
	c.byItem[itemID] = conv
	c.mu.Unlock()

	return conv, nil
}

func (c *ChanceCalculator) globalConversion(ctx context.Context) (conversionRate, error) {
	c.mu.Lock()
	cached := c.global
	c.mu.Unlock()

	if cached.basis != "" && time.Now().Before(cached.expiresAt) {
		return cached, nil
	}

	purchased, expired, err := c.repo.OutcomesTotal(ctx)
	if err != nil {
		return conversionRate{}, fmt.Errorf("counting total outcomes: %w", err)
	}

	conv := conversionRate{
		value:     conversionDefault,
		basis:     conversionBasisDefault,
		expiresAt: time.Now().Add(conversionTTL),
	}
	if resolved := purchased + expired; resolved > 0 {
		conv.value = float64(purchased) / float64(resolved)
		conv.basis = conversionBasisGlobal
	}

	c.mu.Lock()
	c.global = conv
	c.mu.Unlock()

	return conv, nil
}

// chancePercent — P( Binomial(ahead, conv) < unitsLeft ) в процентах.
// Считается точно по функции распределения: слагаемые выводятся друг из друга
// рекуррентно, факториалы не вычисляются и переполниться нечему.
func chancePercent(ahead, unitsLeft int, conv float64) int {
	switch {
	case unitsLeft <= 0:
		return 0
	case ahead <= 0, unitsLeft > ahead:
		// Даже если всё впереди раскупят, единица останется.
		return 100
	case conv <= 0:
		// Никто впереди не доводит до покупки — слоты вернутся все.
		return 100
	case conv >= 1:
		// Все впереди покупают, а их не меньше остатка.
		return 0
	}

	// term — вероятность ровно i покупок среди тех, кто впереди.
	term := math.Pow(1-conv, float64(ahead))
	cumulative := term

	for i := 0; i < unitsLeft-1; i++ {
		term *= float64(ahead-i) / float64(i+1) * conv / (1 - conv)
		cumulative += term
	}

	return int(math.Round(math.Min(cumulative, 1) * 100))
}

// progressPercent — насколько продвинулась очередь с момента входа.
// nil, когда полосу рисовать не на чем: позиции на входе нет у записей, живших
// до появления колонки, а вошедшему первым двигаться некуда.
func progressPercent(initialPosition *int, position int) *int {
	if initialPosition == nil || *initialPosition <= 1 || position <= 0 {
		return nil
	}

	passed := *initialPosition - position
	if passed < 0 {
		passed = 0
	}

	percent := passed * 100 / (*initialPosition - 1)
	if percent > 100 {
		percent = 100
	}

	return &percent
}
