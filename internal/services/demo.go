package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"avito-queue/internal/domain"
)

type demoQueueRepo interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
	ExpireNow(ctx context.Context, itemID uuid.UUID) error
}

// demoQueueService — часть QueueEntryService, нужная демо-ручкам: Entry для
// simulate (настоящий вход в очередь, не подделка состояния) и Reconcile для
// expire-now (сразу отдать освободившийся слот следующему).
type demoQueueService interface {
	Entry(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueStatusResponse, error)
	Reconcile(ctx context.Context, itemID uuid.UUID) error
}

type demoCatalogRepo interface {
	LockItem(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error)
}

type DemoService struct {
	queueRepo   demoQueueRepo
	catalogRepo demoCatalogRepo
	queue       demoQueueService
}

func NewDemoService(queueRepo demoQueueRepo, catalogRepo demoCatalogRepo, queue demoQueueService) *DemoService {
	return &DemoService{
		queueRepo:   queueRepo,
		catalogRepo: catalogRepo,
		queue:       queue,
	}
}

// ExpireNow двигает expires_at всех выданных прав на товар в прошлое и сразу
// продвигает очередь — без этого демонстрация истечения права занимает
// hold_ttl_seconds реального ожидания.
func (d *DemoService) ExpireNow(ctx context.Context, itemID uuid.UUID) error {
	err := d.queueRepo.InTx(ctx, func(ctx context.Context) error {
		if _, err := d.catalogRepo.LockItem(ctx, itemID); err != nil {
			return fmt.Errorf("locking item: %w", err)
		}

		if err := d.queueRepo.ExpireNow(ctx, itemID); err != nil {
			return fmt.Errorf("expiring purchase rights: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("expiring item rights now: %w", err)
	}

	if err := d.queue.Reconcile(ctx, itemID); err != nil {
		return fmt.Errorf("reconciling after expire-now: %w", err)
	}

	return nil
}

// Simulate конкурентно заводит count пользователей в очередь через настоящий
// QueueService.Entry — доказательство защиты от гонок реальными ручками, а не
// декоративным симулятором поверх мока.
func (d *DemoService) Simulate(ctx context.Context, itemID uuid.UUID, count int) (joined, failed int, err error) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()

			_, entryErr := d.queue.Entry(ctx, uuid.New(), itemID)

			mu.Lock()
			defer mu.Unlock()
			if entryErr != nil {
				failed++
				return
			}
			joined++
		}()
	}

	wg.Wait()

	return joined, failed, nil
}
