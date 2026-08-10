package services

import (
	"avito-queue/internal/domain"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type CatalogRepo interface {
	// LockItem блокирует строку товара первым запросом транзакции (INV-3).
	LockItem(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error)
	AdjustCounts(ctx context.Context, id uuid.UUID, grantedDelta, usedDelta int) error
	GetItemByID(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error)
	GetSimilarItems(ctx context.Context, item domain.CatalogItem) ([]domain.CatalogItem, error)
}

type QueueRepo interface {
	Entry(ctx context.Context, userID, itemID uuid.UUID) error
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
	MarkGrantedExpired(ctx context.Context, itemID uuid.UUID, userIDs []uuid.UUID) (int64, error)
	NeedsReconcile(ctx context.Context, itemID uuid.UUID) (bool, error)
	GetWaiting(ctx context.Context, itemID uuid.UUID, freeSlots int) ([]uuid.UUID, error)
	// UpdateStatus переводит запись из from в to; фильтр по from обязателен —
	// у пользователя может быть несколько записей по товару после повторного входа.
	UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, from, to domain.QueueStatus) error
	GetRecord(ctx context.Context, userID, itemID uuid.UUID) (domain.Queue, error)
	GetPosition(ctx context.Context, queueRecord domain.Queue) (int, error)
	MarkSoldOut(ctx context.Context, itemID uuid.UUID) error
	CountWaiting(ctx context.Context, itemID uuid.UUID) (int, error)
	// Now — время Postgres, а не Go-часов (INV-6): иначе server_time сам
	// становится источником рассинхронизации при дрейфе app/db-контейнера.
	Now(ctx context.Context) (time.Time, error)
}

type PurchaseRightRepo interface {
	Create(ctx context.Context, userID, itemID uuid.UUID) error
	CountGranted(ctx context.Context, itemID uuid.UUID) (int, error)
	CountUsed(ctx context.Context, itemID uuid.UUID) (int, error)
	UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, from, to domain.PurchaseRightStatus) error
	ExpireOld(ctx context.Context, itemID uuid.UUID) ([]uuid.UUID, error)
	GetByUserAndItem(ctx context.Context, userID, itemID uuid.UUID) (domain.PurchaseRight, error)
}

type QueueService struct {
	QueueRepo         QueueRepo
	PurchaseRightRepo PurchaseRightRepo
	CatalogRepo       CatalogRepo
	logger            *slog.Logger
}

func NewQueueService(queueRepo QueueRepo, purchaseRightRepo PurchaseRightRepo, catalogRepo CatalogRepo, logger *slog.Logger) *QueueService {
	return &QueueService{
		QueueRepo:         queueRepo,
		PurchaseRightRepo: purchaseRightRepo,
		CatalogRepo:       catalogRepo,
		logger:            logger,
	}
}

// statusPresentation — message и next_step на каждое состояние очереди (INV-8).
var statusPresentation = map[domain.QueueStatus]struct {
	Message  string
	NextStep domain.NextStep
}{
	domain.QueueStatusNotInQueue: {
		Message:  "Вы ещё не в очереди на этот товар.",
		NextStep: domain.NextStep{Kind: "join", Label: "Встаньте в очередь, чтобы получить право на покупку"},
	},
	domain.QueueStatusWaiting: {
		Message:  "Вы в очереди. Пожалуйста, подождите освобождения слота.",
		NextStep: domain.NextStep{Kind: "wait", Label: "Ожидайте, статус обновится сам"},
	},
	domain.QueueStatusGranted: {
		Message:  "Ваша очередь подошла! У вас есть несколько минут на оплату.",
		NextStep: domain.NextStep{Kind: "pay", Label: "Оплатите, пока не истекло время"},
	},
	domain.QueueStatusPurchased: {
		Message:  "Покупка успешно завершена.",
		NextStep: domain.NextStep{Kind: "done", Label: "Покупка завершена"},
	},
	domain.QueueStatusExpired: {
		Message:  "Время вышло, право на покупку сгорело. Вы можете встать в очередь заново.",
		NextStep: domain.NextStep{Kind: "rejoin", Label: "Встаньте в очередь заново"},
	},
	domain.QueueStatusSoldOut: {
		Message:  "К сожалению, товар закончился. Посмотрите похожие лоты.",
		NextStep: domain.NextStep{Kind: "browse_similar", Label: "Посмотрите похожие товары"},
	},
	domain.QueueStatusCancelled: {
		Message:  "Вы покинули очередь.",
		NextStep: domain.NextStep{Kind: "rejoin", Label: "Встаньте в очередь заново, если товар ещё нужен"},
	},
}

// Entry — постановка в очередь, идемпотентная по паре пользователь-товар
// (QUE-02, A-08): повторный вход не ошибка, а тот же конверт статуса, что и
// GetStatus.
func (q *QueueService) Entry(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueStatusResponse, error) {
	if _, err := q.CatalogRepo.GetItemByID(ctx, itemID); err != nil {
		return domain.QueueStatusResponse{}, fmt.Errorf("checking item exists: %w", err)
	}

	if err := q.QueueRepo.Entry(ctx, userID, itemID); err != nil && !errors.Is(err, domain.ErrUserAlreadyInQueue) {
		return domain.QueueStatusResponse{}, fmt.Errorf("adding user to queue: %w", err)
	}

	if err := q.EnsureAdvanced(ctx, itemID); err != nil {
		return domain.QueueStatusResponse{}, fmt.Errorf("reconciling queue: %w", err)
	}

	return q.buildStatusResponse(ctx, userID, itemID)
}

// EnsureAdvanced продвигает очередь товара, только если есть работа: без
// фонового воркера Reconcile двигается синхронно из любой ручки, и
// NeedsReconcile — дешёвая проверка без блокировки строки, чтобы частый
// поллинг статуса не превращал её в горячую точку.
func (q *QueueService) EnsureAdvanced(ctx context.Context, itemID uuid.UUID) error {
	need, err := q.QueueRepo.NeedsReconcile(ctx, itemID)
	if err != nil {
		return fmt.Errorf("checking if reconcile is needed: %w", err)
	}
	if !need {
		return nil
	}

	return q.Reconcile(ctx, itemID)
}

// Reconcile — единственное место (вместе с PurchaseRight.Buy), которое меняет
// распределение прав (INV-2): гасит просроченные права, возвращает слоты в
// пул и раздаёт их первым в очереди. Одна транзакция с блокировкой товара
// первым запросом (INV-3).
func (q *QueueService) Reconcile(ctx context.Context, itemID uuid.UUID) error {
	return q.QueueRepo.InTx(ctx, func(ctx context.Context) error {
		item, err := q.CatalogRepo.LockItem(ctx, itemID)
		if err != nil {
			return fmt.Errorf("locking item: %w", err)
		}

		expiredUserIDs, err := q.PurchaseRightRepo.ExpireOld(ctx, itemID)
		if err != nil {
			return fmt.Errorf("expiring old purchase rights: %w", err)
		}

		rowsAffected, err := q.QueueRepo.MarkGrantedExpired(ctx, itemID, expiredUserIDs)
		if err != nil {
			return fmt.Errorf("expiring queue records: %w", err)
		}
		if rowsAffected != int64(len(expiredUserIDs)) {
			q.logger.Error("queue/purchase_rights granted state mismatch",
				"item_id", itemID, "expired_rights", len(expiredUserIDs), "expired_queue_rows", rowsAffected)
		}

		if len(expiredUserIDs) > 0 {
			if err := q.CatalogRepo.AdjustCounts(ctx, itemID, -len(expiredUserIDs), 0); err != nil {
				return fmt.Errorf("adjusting stock counters after expiry: %w", err)
			}
		}

		activeRights, err := q.PurchaseRightRepo.CountGranted(ctx, itemID)
		if err != nil {
			return fmt.Errorf("counting active purchase rights: %w", err)
		}

		usedRights, err := q.PurchaseRightRepo.CountUsed(ctx, itemID)
		if err != nil {
			return fmt.Errorf("counting used purchase rights: %w", err)
		}

		if usedRights >= item.TotalStock {
			if err := q.QueueRepo.MarkSoldOut(ctx, item.ID); err != nil {
				return fmt.Errorf("marking soldOut: %w", err)
			}

			return nil
		}

		possibleRights := item.TotalStock - activeRights - usedRights
		if possibleRights < 0 {
			q.logger.Error("purchase rights is negative",
				"item_id", itemID, "total_stock", item.TotalStock, "active_rights", activeRights, "used_rights", usedRights)
			return fmt.Errorf("the amount the active rights is larger than item total stock")
		}

		userIDs, err := q.QueueRepo.GetWaiting(ctx, itemID, possibleRights)
		if err != nil {
			return fmt.Errorf("getting waiting users from queue: %w", err)
		}

		for _, userID := range userIDs {
			if err := q.PurchaseRightRepo.Create(ctx, userID, itemID); err != nil {
				return fmt.Errorf("creating purchase right for user: %w", err)
			}

			if err := q.QueueRepo.UpdateStatus(ctx, userID, itemID, domain.QueueStatusWaiting, domain.QueueStatusGranted); err != nil {
				return fmt.Errorf("updating user status: %w", err)
			}
		}

		if len(userIDs) > 0 {
			if err := q.CatalogRepo.AdjustCounts(ctx, itemID, len(userIDs), 0); err != nil {
				return fmt.Errorf("adjusting stock counters after grant: %w", err)
			}
		}

		return nil
	})
}

func (q *QueueService) GetPosition(ctx context.Context, userID, itemID uuid.UUID) (int, error) {
	queueRecord, err := q.QueueRepo.GetRecord(ctx, userID, itemID)
	if err != nil {
		return 0, fmt.Errorf("getting queue record: %w", err)
	}

	pos, err := q.QueueRepo.GetPosition(ctx, queueRecord)
	if err != nil {
		return 0, fmt.Errorf("getting position: %w", err)
	}

	return pos, nil
}

// GetStatus — единый конверт статуса для GET /catalog/:id/queue/me.
func (q *QueueService) GetStatus(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueStatusResponse, error) {
	if err := q.EnsureAdvanced(ctx, itemID); err != nil {
		return domain.QueueStatusResponse{}, fmt.Errorf("ensuring queue advanced: %w", err)
	}

	return q.buildStatusResponse(ctx, userID, itemID)
}

func (q *QueueService) buildStatusResponse(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueStatusResponse, error) {
	// Отсутствие записи — тоже состояние (not_in_queue), а не 404 (INV-8).
	status := domain.QueueStatusNotInQueue

	record, err := q.QueueRepo.GetRecord(ctx, userID, itemID)
	switch {
	case err == nil:
		status = record.Status
	case !errors.Is(err, domain.ErrUserNotFound):
		return domain.QueueStatusResponse{}, fmt.Errorf("getting queue record: %w", err)
	}

	resp := domain.QueueStatusResponse{
		Status:       status,
		Alternatives: []uuid.UUID{},
	}

	if presentation, ok := statusPresentation[status]; ok {
		resp.Message = presentation.Message
		resp.NextStep = presentation.NextStep
	}

	switch status {
	case domain.QueueStatusWaiting:
		resp.Position, err = q.QueueRepo.GetPosition(ctx, record)
		if err != nil {
			return domain.QueueStatusResponse{}, fmt.Errorf("getting position: %w", err)
		}
	case domain.QueueStatusGranted:
		right, rightErr := q.PurchaseRightRepo.GetByUserAndItem(ctx, userID, itemID)
		if rightErr != nil {
			return domain.QueueStatusResponse{}, fmt.Errorf("getting purchase right: %w", rightErr)
		}
		expiresAtValue := right.ExpiresAt
		resp.ExpiresAt = &expiresAtValue
	case domain.QueueStatusSoldOut, domain.QueueStatusExpired:
		resp.Alternatives, err = q.alternatives(ctx, itemID)
		if err != nil {
			return domain.QueueStatusResponse{}, fmt.Errorf("getting alternatives: %w", err)
		}
	}

	resp.QueueSize, err = q.QueueRepo.CountWaiting(ctx, itemID)
	if err != nil {
		return domain.QueueStatusResponse{}, fmt.Errorf("counting queue size: %w", err)
	}

	resp.ServerTime, err = q.QueueRepo.Now(ctx)
	if err != nil {
		return domain.QueueStatusResponse{}, fmt.Errorf("getting server time: %w", err)
	}

	return resp, nil
}

// alternatives — похожие лоты для sold_out/expired, только для них: остальные
// статусы поллятся раз в секунду и в альтернативах не нуждаются. Распроданные
// лоты отсеивает сам GetSimilarItems.
func (q *QueueService) alternatives(ctx context.Context, itemID uuid.UUID) ([]uuid.UUID, error) {
	item, err := q.CatalogRepo.GetItemByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("getting item: %w", err)
	}

	similar, err := q.CatalogRepo.GetSimilarItems(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("getting similar items: %w", err)
	}

	ids := make([]uuid.UUID, 0, len(similar))
	for _, alternative := range similar {
		ids = append(ids, alternative.ID)
	}

	return ids, nil
}

// Leave — выход из очереди. Если было выдано право, оно гасится в той же
// транзакции, что и запись очереди (INV-5). Reconcile после коммита отдаёт
// освободившийся слот следующему сразу, а не при следующем случайном опросе.
func (q *QueueService) Leave(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueStatusResponse, error) {
	err := q.QueueRepo.InTx(ctx, func(ctx context.Context) error {
		if _, err := q.CatalogRepo.LockItem(ctx, itemID); err != nil {
			return fmt.Errorf("locking item: %w", err)
		}

		record, err := q.QueueRepo.GetRecord(ctx, userID, itemID)
		if err != nil {
			return fmt.Errorf("getting queue record: %w", err)
		}

		if record.Status != domain.QueueStatusWaiting && record.Status != domain.QueueStatusGranted {
			return domain.ErrCannotLeaveQueue
		}

		if record.Status == domain.QueueStatusGranted {
			if err := q.PurchaseRightRepo.UpdateStatus(ctx, userID, itemID,
				domain.PurchaseRightStatusGranted, domain.PurchaseRightStatusCancelled); err != nil {
				return fmt.Errorf("cancelling purchase right: %w", err)
			}

			if err := q.CatalogRepo.AdjustCounts(ctx, itemID, -1, 0); err != nil {
				return fmt.Errorf("adjusting stock counters: %w", err)
			}
		}

		if err := q.QueueRepo.UpdateStatus(ctx, userID, itemID, record.Status, domain.QueueStatusCancelled); err != nil {
			return fmt.Errorf("cancelling queue record: %w", err)
		}

		return nil
	})
	if err != nil {
		return domain.QueueStatusResponse{}, fmt.Errorf("leaving queue: %w", err)
	}

	if err := q.Reconcile(ctx, itemID); err != nil {
		return domain.QueueStatusResponse{}, fmt.Errorf("reconciling after leave: %w", err)
	}

	return q.buildStatusResponse(ctx, userID, itemID)
}
