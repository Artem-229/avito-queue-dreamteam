package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"avito-queue/internal/domain"
)

type queueEntriesRepository interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
	Enter(ctx context.Context, userID, itemID uuid.UUID) error
	ExpireOverdue(ctx context.Context, itemID uuid.UUID) ([]uuid.UUID, error)
	GrantNext(ctx context.Context, itemID uuid.UUID, slots int) ([]uuid.UUID, error)
	MarkPurchased(ctx context.Context, userID, itemID uuid.UUID) (bool, error)
	UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, from, to domain.QueueEntryStatus) error
	MarkSoldOut(ctx context.Context, itemID uuid.UUID) error
	NeedsReconcile(ctx context.Context, itemID uuid.UUID) (bool, error)
	CountByStatus(ctx context.Context, itemID uuid.UUID, status domain.QueueEntryStatus) (int, error)
	GetEntry(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueEntry, error)
	Snapshot(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueSnapshot, error)
}

// catalogRepository — каталог глазами очереди. AdjustCounts здесь намеренно
// нет: счётчики товара ведёт триггер queue_entries_sync_counters, и ручная
// арифметика распределения в Go больше не существует.
type catalogRepository interface {
	// LockItem блокирует строку товара первым запросом транзакции (INV-3).
	LockItem(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error)
	GetItemByID(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error)
	GetSimilarItems(ctx context.Context, id uuid.UUID) ([]domain.CatalogItem, error)
}

type chanceCalculator interface {
	For(ctx context.Context, itemID uuid.UUID, position, grantedCount, usedCount, totalStock int) (percent int, basis string, err error)
}

type QueueEntryService struct {
	QueueRepo   queueEntriesRepository
	CatalogRepo catalogRepository
	Chance      chanceCalculator
}

func NewQueueEntryService(queueRepo queueEntriesRepository, catalogRepo catalogRepository, chance chanceCalculator) *QueueEntryService {
	return &QueueEntryService{
		QueueRepo:   queueRepo,
		CatalogRepo: catalogRepo,
		Chance:      chance,
	}
}

// entryPresentation — message и next_step на каждое состояние (INV-8).
// Серых зон, где пользователю непонятно, что происходит и что делать дальше,
// в API нет: состояние без сообщения и следующего шага не существует.
var entryPresentation = map[domain.QueueEntryStatus]struct {
	Message  string
	NextStep domain.NextStep
}{
	domain.QueueEntryStatusNotInQueue: {
		Message:  "Вы ещё не в очереди на этот товар.",
		NextStep: domain.NextStep{Kind: "join", Label: "Встаньте в очередь, чтобы получить право на покупку"},
	},
	domain.QueueEntryStatusWaiting: {
		Message:  "Вы в очереди. Пожалуйста, подождите освобождения слота.",
		NextStep: domain.NextStep{Kind: "wait", Label: "Ожидайте, статус обновится сам"},
	},
	domain.QueueEntryStatusGranted: {
		Message:  "Ваша очередь подошла! У вас есть несколько минут на оплату.",
		NextStep: domain.NextStep{Kind: "pay", Label: "Оплатите, пока не истекло время"},
	},
	domain.QueueEntryStatusPurchased: {
		Message:  "Покупка успешно завершена.",
		NextStep: domain.NextStep{Kind: "done", Label: "Покупка завершена"},
	},
	domain.QueueEntryStatusExpired: {
		Message:  "Время вышло, право на покупку сгорело. Вы можете встать в очередь заново.",
		NextStep: domain.NextStep{Kind: "rejoin", Label: "Встаньте в очередь заново"},
	},
	domain.QueueEntryStatusSoldOut: {
		Message:  "К сожалению, товар закончился. Посмотрите похожие лоты.",
		NextStep: domain.NextStep{Kind: "browse_similar", Label: "Посмотрите похожие товары"},
	},
	domain.QueueEntryStatusCancelled: {
		Message:  "Вы покинули очередь.",
		NextStep: domain.NextStep{Kind: "rejoin", Label: "Встаньте в очередь заново, если товар ещё нужен"},
	},
}

// Entry — постановка в очередь. Идемпотентна: повторный вход не ошибка, а тот
// же конверт состояния, что вернул бы GetStatus.
func (q *QueueEntryService) Entry(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueStatusResponse, error) {
	if _, err := q.CatalogRepo.GetItemByID(ctx, itemID); err != nil {
		return domain.QueueStatusResponse{}, fmt.Errorf("checking item exists: %w", err)
	}

	if err := q.QueueRepo.Enter(ctx, userID, itemID); err != nil && !errors.Is(err, domain.ErrUserAlreadyInQueue) {
		return domain.QueueStatusResponse{}, fmt.Errorf("entering queue: %w", err)
	}

	if err := q.EnsureAdvanced(ctx, itemID); err != nil {
		return domain.QueueStatusResponse{}, fmt.Errorf("advancing queue: %w", err)
	}

	return q.Status(ctx, userID, itemID)
}

// Status — конверт состояния для GET /catalog/:id/queue/me. Эту ручку фронт
// опрашивает раз в секунду, она же продвигает очередь: фоновых воркеров нет.
func (q *QueueEntryService) Status(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueStatusResponse, error) {
	if err := q.EnsureAdvanced(ctx, itemID); err != nil {
		return domain.QueueStatusResponse{}, fmt.Errorf("advancing queue: %w", err)
	}

	snapshot, err := q.QueueRepo.Snapshot(ctx, userID, itemID)
	if err != nil {
		return domain.QueueStatusResponse{}, fmt.Errorf("getting snapshot: %w", err)
	}

	return q.buildResponse(ctx, snapshot, itemID)
}

// Leave — выход из очереди. Выданное право гасится вместе с записью: это одна
// строка, гасить в двух местах больше нечего. Reconcile после коммита отдаёт
// освободившийся слот следующему сразу, а не при следующем случайном опросе.
func (q *QueueEntryService) Leave(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueStatusResponse, error) {
	err := q.QueueRepo.InTx(ctx, func(ctx context.Context) error {
		if _, err := q.CatalogRepo.LockItem(ctx, itemID); err != nil {
			return fmt.Errorf("locking item: %w", err)
		}

		entry, err := q.QueueRepo.GetEntry(ctx, userID, itemID)
		if err != nil {
			return fmt.Errorf("getting entry: %w", err)
		}

		if entry.Status != domain.QueueEntryStatusWaiting && entry.Status != domain.QueueEntryStatusGranted {
			return domain.ErrCannotLeaveQueue
		}

		if err := q.QueueRepo.UpdateStatus(ctx, userID, itemID,
			entry.Status, domain.QueueEntryStatusCancelled); err != nil {
			return fmt.Errorf("cancelling entry: %w", err)
		}

		return nil
	})
	if err != nil {
		return domain.QueueStatusResponse{}, fmt.Errorf("leaving queue: %w", err)
	}

	if err := q.Reconcile(ctx, itemID); err != nil {
		return domain.QueueStatusResponse{}, fmt.Errorf("reconciling after leave: %w", err)
	}

	return q.Status(ctx, userID, itemID)
}

// Buy тратит выданное право — вместе с Reconcile единственное место, меняющее
// распределение (INV-2). Продвижение очереди перед покупкой обязательно: иначе
// Buy мог бы увидеть уже истёкшее, но ещё не погашенное право.
func (q *QueueEntryService) Buy(ctx context.Context, userID, itemID uuid.UUID) error {
	if err := q.EnsureAdvanced(ctx, itemID); err != nil {
		return fmt.Errorf("advancing queue: %w", err)
	}

	return q.QueueRepo.InTx(ctx, func(ctx context.Context) error {
		item, err := q.CatalogRepo.LockItem(ctx, itemID)
		if err != nil {
			return fmt.Errorf("locking item: %w", err)
		}

		// Проверка права и его трата — один UPDATE: между «посмотрели» и
		// «потратили» не остаётся окна, в котором право успело бы истечь или
		// уйти повторному запросу того же пользователя.
		bought, err := q.QueueRepo.MarkPurchased(ctx, userID, itemID)
		if err != nil {
			return fmt.Errorf("marking purchased: %w", err)
		}
		if !bought {
			return q.noRightError(ctx, item)
		}

		return nil
	})
}

// EnsureAdvanced продвигает очередь товара, только если есть работа.
// NeedsReconcile — дешёвая проверка без блокировки строки: поллинг раз в
// секунду не должен превращать строку товара в горячую точку.
func (q *QueueEntryService) EnsureAdvanced(ctx context.Context, itemID uuid.UUID) error {
	need, err := q.QueueRepo.NeedsReconcile(ctx, itemID)
	if err != nil {
		return fmt.Errorf("checking if reconcile is needed: %w", err)
	}
	if !need {
		return nil
	}

	return q.Reconcile(ctx, itemID)
}

// Reconcile гасит просроченные права, возвращает освободившиеся слоты в пул и
// раздаёт их первым в очереди. Одна транзакция, блокировка товара первым
// запросом (INV-3): все операции по одному товару сериализуются, по разным —
// не мешают друг другу.
func (q *QueueEntryService) Reconcile(ctx context.Context, itemID uuid.UUID) error {
	return q.QueueRepo.InTx(ctx, func(ctx context.Context) error {
		item, err := q.CatalogRepo.LockItem(ctx, itemID)
		if err != nil {
			return fmt.Errorf("locking item: %w", err)
		}

		if _, err := q.QueueRepo.ExpireOverdue(ctx, itemID); err != nil {
			return fmt.Errorf("expiring overdue rights: %w", err)
		}

		// Считаем по queue_entries, а не по счётчикам прочитанного товара:
		// их обновил триггер уже после того, как LockItem вернул строку.
		used, err := q.QueueRepo.CountByStatus(ctx, itemID, domain.QueueEntryStatusPurchased)
		if err != nil {
			return fmt.Errorf("counting purchased: %w", err)
		}

		// Весь тираж выкуплен: свободных слотов нет и не появится, ожидающим
		// незачем висеть в waiting. Держателей права в этот момент не
		// существует — granted + used <= total_stock при used = total_stock.
		if used >= item.TotalStock {
			if err := q.QueueRepo.MarkSoldOut(ctx, itemID); err != nil {
				return fmt.Errorf("marking sold_out: %w", err)
			}

			return nil
		}

		granted, err := q.QueueRepo.CountByStatus(ctx, itemID, domain.QueueEntryStatusGranted)
		if err != nil {
			return fmt.Errorf("counting granted: %w", err)
		}

		slots := item.TotalStock - used - granted
		if slots <= 0 {
			return nil
		}

		if _, err := q.QueueRepo.GrantNext(ctx, itemID, slots); err != nil {
			return fmt.Errorf("granting rights: %w", err)
		}

		return nil
	})
}

// buildResponse собирает конверт состояния из снимка: одно место, где
// состояние превращается в то, что видит пользователь. Все ручки очереди
// отвечают одинаково — экрану не приходится склеивать состояние из разных
// форматов.
func (q *QueueEntryService) buildResponse(ctx context.Context, snapshot domain.QueueSnapshot, itemID uuid.UUID) (domain.QueueStatusResponse, error) {
	status := snapshot.Status()

	resp := domain.QueueStatusResponse{
		Status:       status,
		QueueSize:    snapshot.QueueSize,
		ServerTime:   snapshot.ServerTime,
		Alternatives: []uuid.UUID{},
	}

	if presentation, ok := entryPresentation[status]; ok {
		resp.Message = presentation.Message
		resp.NextStep = presentation.NextStep
	}

	switch status {
	case domain.QueueEntryStatusWaiting:
		resp.Position = snapshot.Position
		resp.InitialPosition = snapshot.Entry.InitialPosition
		resp.ProgressPercent = progressPercent(snapshot.Entry.InitialPosition, snapshot.Position)

		percent, basis, err := q.Chance.For(ctx, itemID,
			snapshot.Position, snapshot.GrantedCount, snapshot.UsedCount, snapshot.TotalStock)
		if err != nil {
			return domain.QueueStatusResponse{}, fmt.Errorf("calculating chance: %w", err)
		}
		resp.Chance = &domain.PurchaseChance{Percent: percent, Basis: basis}

	case domain.QueueEntryStatusGranted:
		resp.ExpiresAt = snapshot.Entry.ExpiresAt

	case domain.QueueEntryStatusSoldOut, domain.QueueEntryStatusExpired:
		alternatives, err := q.alternatives(ctx, itemID)
		if err != nil {
			return domain.QueueStatusResponse{}, fmt.Errorf("getting alternatives: %w", err)
		}
		resp.Alternatives = alternatives
	}

	return resp, nil
}

// alternatives — похожие лоты для состояний, где текущий товар уже не получить.
// Пока читаются напрямую из каталога; когда появится кэш рекомендаций (REC-04),
// поменяется только источник — контракт ответа тот же.
func (q *QueueEntryService) alternatives(ctx context.Context, itemID uuid.UUID) ([]uuid.UUID, error) {
	_, err := q.CatalogRepo.GetItemByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("getting item: %w", err)
	}

	similar, err := q.CatalogRepo.GetSimilarItems(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("getting similar items: %w", err)
	}

	ids := make([]uuid.UUID, 0, len(similar))
	for _, alternative := range similar {
		ids = append(ids, alternative.ID)
	}

	return ids, nil
}

// noRightError уточняет причину отказа: если весь тираж разобран, пользователю
// нужно сказать «товар закончился», а не «у вас нет права» — это разные экраны
// и разные следующие шаги.
func (q *QueueEntryService) noRightError(ctx context.Context, item domain.CatalogItem) error {
	used, err := q.QueueRepo.CountByStatus(ctx, item.ID, domain.QueueEntryStatusPurchased)
	if err != nil {
		return domain.ErrNoPurchaseRight
	}

	if used >= item.TotalStock {
		return domain.ErrItemSoldOut
	}

	return domain.ErrNoPurchaseRight
}
