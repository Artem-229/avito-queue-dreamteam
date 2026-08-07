package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"avito-queue/internal/domain"
)

type mockRepo struct {
	id        uuid.UUID
	status    domain.PurchaseStatus
	expiresAt time.Time
	findErr   error

	markSuccess bool
	markErr     error
}

func (m mockRepo) FindByUserAndItem(ctx context.Context, userID, itemID uuid.UUID) (uuid.UUID, domain.PurchaseStatus, time.Time, error) {
	return m.id, m.status, m.expiresAt, m.findErr
}

func (m mockRepo) MarkAsUsed(ctx context.Context, purchaseID uuid.UUID) (bool, error) {
	return m.markSuccess, m.markErr
}

// Право активно. happy path
func TestCheckAccess_Granted(t *testing.T) {
	id := uuid.New()
	repo := mockRepo{
		id:        id,
		status:    domain.StatusGranted,
		expiresAt: time.Now().Add(time.Hour),
	}
	usecase := NewCheckoutUsecase(repo)

	purchaseID, _, allowed, _, err := usecase.CheckAccess(context.Background(), uuid.New(), uuid.New())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected allowed=true for granted, non-expired right")
	}
	if purchaseID != id {
		t.Errorf("expected purchaseID=%v, got %v", id, purchaseID)
	}
}

// Право истекло
func TestCheckAccess_Expired(t *testing.T) {
	repo := mockRepo{
		status:    domain.StatusGranted,
		expiresAt: time.Now().Add(-time.Hour),
	}
	usecase := NewCheckoutUsecase(repo)

	_, _, allowed, reason, err := usecase.CheckAccess(context.Background(), uuid.New(), uuid.New())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected allowed=false for expired right")
	}
	if reason == "" {
		t.Error("expected non-empty reason for expired right")
	}
}

// Право уже использовано
func TestCheckAccess_NotGranted(t *testing.T) {
	repo := mockRepo{
		status:    domain.StatusUsed,
		expiresAt: time.Now().Add(time.Hour),
	}
	usecase := NewCheckoutUsecase(repo)

	_, _, allowed, _, err := usecase.CheckAccess(context.Background(), uuid.New(), uuid.New())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected allowed=false for status=used")
	}
}

// Права не существовало
func TestCheckAccess_NoRows(t *testing.T) {
	repo := mockRepo{
		findErr: pgx.ErrNoRows,
	}
	usecase := NewCheckoutUsecase(repo)

	_, _, allowed, reason, err := usecase.CheckAccess(context.Background(), uuid.New(), uuid.New())

	if err != nil {
		t.Fatalf("expected nil error for no-rows case, got: %v", err)
	}
	if allowed {
		t.Error("expected allowed=false when no right exists")
	}
	if reason == "" {
		t.Error("expected non-empty reason when no right exists")
	}
}

// Сбой в работе БД
func TestCheckAccess_DBError(t *testing.T) {
	repo := mockRepo{
		findErr: errors.New("connection refused"),
	}
	usecase := NewCheckoutUsecase(repo)

	_, _, allowed, _, err := usecase.CheckAccess(context.Background(), uuid.New(), uuid.New())

	if err == nil {
		t.Fatal("expected non-nil error for db failure")
	}
	if allowed {
		t.Error("expected allowed=false on db error")
	}
}

// Успешная оплата
func TestPay_Success(t *testing.T) {
	repo := mockRepo{markSuccess: true}
	usecase := NewCheckoutUsecase(repo)

	success, _, err := usecase.Pay(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !success {
		t.Error("expected success=true")
	}
}

// Неуспешная оплата, право истекло
func TestPay_AlreadyUsedOrExpired(t *testing.T) {
	repo := mockRepo{markSuccess: false}
	usecase := NewCheckoutUsecase(repo)

	success, reason, err := usecase.Pay(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if success {
		t.Error("expected success=false when 0 rows affected")
	}
	if reason == "" {
		t.Error("expected non-empty reason on failed purchase")
	}
}

// Ошибка БД
func TestPay_DBError(t *testing.T) {
	repo := mockRepo{markErr: errors.New("connection refused")}
	usecase := NewCheckoutUsecase(repo)

	success, _, err := usecase.Pay(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("expected non-nil error for db failure")
	}
	if success {
		t.Error("expected success=false on db error")
	}
}
