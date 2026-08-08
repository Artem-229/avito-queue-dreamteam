package services

import (
	"avito-queue/internal/domain"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type CatalogRepo interface {
	GetItemByID(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error)
}

type QueueRepo interface {
	Entry(ctx context.Context, userID, itemID uuid.UUID) error
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
	MarkRecordsExpired(ctx context.Context, itemID uuid.UUID, userIDs []uuid.UUID) error
	GetWaiting(ctx context.Context, itemID uuid.UUID, freeSlots int) ([]uuid.UUID, error)
	UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, status domain.QueueStatus) error
	GetRecord(ctx context.Context, userID, itemID uuid.UUID) (domain.Queue, error)
	GetPosition(ctx context.Context, queueRecord domain.Queue) (int, error)
	MarkSoldOut(ctx context.Context, itemID uuid.UUID) error
}

type PurchaseRightRepo interface {
	Create(ctx context.Context, userID, itemID uuid.UUID) error
	CountGranted(ctx context.Context, itemID uuid.UUID) (int, error)
	CountUsed(ctx context.Context, itemID uuid.UUID) (int, error)
	UpdateStatus(ctx context.Context, userID, itemID uuid.UUID, status domain.PurchaseRightStatus) error
	ExpireOld(ctx context.Context, itemID uuid.UUID, t time.Time) ([]uuid.UUID, error)
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

func (q *QueueService) Entry(ctx context.Context, userID, itemID uuid.UUID) error {
	if err := q.QueueRepo.Entry(ctx, userID, itemID); err != nil {
		return fmt.Errorf("adding user to queue: %w", err)
	}

	if err := q.Reconcile(ctx, itemID); err != nil {
		return fmt.Errorf("reconcilling queue: %w", err)
	}

	return nil
}

func (q *QueueService) Reconcile(ctx context.Context, itemID uuid.UUID) error {
	// Метод сдвига очереди, включает в себя все шаги
	// Когда человек вступает в очередь он получает себе запись, после этого вызывается сдвиг очереди
	// Сначала происходит пометка просроченных прав чтобы получить список актульных прав на данный момент
	// потом сдвиг очереди получает из каталога предмета его сток а из таблицы прав количество активных на данный момент прав покупки
	// так он считает общее количество прав которое еще можно выдать. После этого он кладет это число в метод выдать права
	// этот метод в таблице очереди для кол-ва людей равного кол-ву прав выдает им права и тд
	// 1) Сброс устаревших прав и обновление статусов (например если человек покинул очередь и его статус сменился надо отменить его право)
	// 2) Пересчет актуального количества прав
	// 3) Выдача прав новым людям
	// Метод вызывается после дергания каждой ручки на сервисе чтобы что угодно сдвигало очередь, таким образом пока хоть один человек стоит в очереди она будет работать
	// Если больше человек чем доступных прав одновременно нажмут купить товар то они все поместятся  в очередь, реконсиляция обнулит просроченные и выдаст права на покупку ровно тому
	// количеству человек, которым хватит. Все это выполняется в одной транзакции и ручки изменения статуса вызываются только отсюда чтобы исключить гонки и неконсистентность данных
	// Человек если хочет например покинуть очередь нажимает на кнопку выйти и его статус в очереди помечается как отмененный после чего сдвиг очереди уже сам сбросит его права
	return q.QueueRepo.InTx(ctx, func(ctx context.Context) error {
		t := time.Now()

		item, err := q.CatalogRepo.GetItemByID(ctx, itemID)
		if err != nil {
			return fmt.Errorf("getting stock: %w", err)
		}

		ids, err := q.PurchaseRightRepo.ExpireOld(ctx, itemID, t)
		if err != nil {
			return fmt.Errorf("expiring old purchase rights: %w", err)
		}

		if err := q.QueueRepo.MarkRecordsExpired(ctx, itemID, ids); err != nil {
			return fmt.Errorf("expiring queue records: %w", err)
		}

		activeRights, err := q.PurchaseRightRepo.CountGranted(ctx, itemID)
		if err != nil {
			return fmt.Errorf("counting active purchase rights: %w", err)
		}

		usedRights, err := q.PurchaseRightRepo.CountUsed(ctx, itemID)
		if err != nil {
			return fmt.Errorf("counting used purchase rights: %w", err)
		}

		if item.TotalStock == usedRights {
			if err := q.QueueRepo.MarkSoldOut(ctx, item.ID); err != nil {
				return fmt.Errorf("marking soldOut: %w", err)
			}

			return domain.ErrItemSoldOut
		}

		possibleRights := item.TotalStock - activeRights - usedRights
		if possibleRights < 0 {
			q.logger.Error("purchase rights is negative", "total stock", item.TotalStock, "activeRights", item.TotalStock)
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

			if err := q.QueueRepo.UpdateStatus(ctx, userID, itemID, domain.QueueStatusGranted); err != nil {
				return fmt.Errorf("updating user status: %w", err)
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
