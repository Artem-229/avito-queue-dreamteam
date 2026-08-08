package repository

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"avito-queue/internal/domain"
)

// Тест на то, применится ли used к праву ровно один раз и не переиспользуется ли право, когда оно уже стало used, при условии ста парралельных попыток применить это право
func TestMarkAsUsed_Concurrency(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/avito_queue?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	itemID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO catalog_items (id, name, price, total_stock, created_at) VALUES ($1, $2, $3, $4, now())`,
		itemID, "concurrency test item", 100.0, 1)
	if err != nil {
		t.Fatalf("insert test catalog item: %v", err)
	}
	defer pool.Exec(ctx, `DELETE FROM catalog_items WHERE id = $1`, itemID)

	userID := uuid.New()
	purchaseID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO purchase_rights (id, item_id, user_id, status, expires_at) VALUES ($1, $2, $3, $4, $5)`,
		purchaseID, itemID, userID, domain.PurchaseRightStatusGranted, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("insert test purchase right: %v", err)
	}
	defer pool.Exec(ctx, `DELETE FROM purchase_rights WHERE id = $1`, purchaseID)

	repo := NewPurchaseRightRepo(pool)

	const n = 100
	var successCount int64
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			success, err := repo.MarkAsUsed(ctx, userID, itemID)
			if err != nil {
				t.Errorf("MarkAsUsed error: %v", err)
				return
			}
			if success {
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}
	wg.Wait()

	if successCount != 1 {
		t.Errorf("expected exactly 1 successful purchase out of %d concurrent attempts, got %d", n, successCount)
	}
}
