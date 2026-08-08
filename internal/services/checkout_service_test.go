package services

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
	id          uuid.UUID
	RightStatus domain.PurchaseRightStatus
	expiresAt   time.Time
	findErr     error

	markSuccess bool
	markErr     error
}

func (m mockRepo) GetByUserAndItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (domain.PurchaseRight, error) {
	return domain.PurchaseRight{
		ID:        m.id,
		UserID:    userID,
		ItemID:    itemID,
		Status:    m.RightStatus,
		ExpiresAt: m.expiresAt,
	}, m.findErr
}

func (m mockRepo) Buy(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) error {
	return nil
}

func (m mockRepo) MarkAsUsed(ctx context.Context, purchaseID uuid.UUID) (bool, error) {
	return m.markSuccess, m.markErr
}

// Право активно. happy path
func TestCheckAccess_Granted(t *testing.T) {
	id := uuid.New()
	repo := mockRepo{
		id:          id,
		RightStatus: domain.PurchaseRightStatusGranted,
		expiresAt:   time.Now().Add(time.Hour),
	}
	usecase := NewCheckoutService(repo, repo)

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
		RightStatus: domain.PurchaseRightStatusGranted,
		expiresAt:   time.Now().Add(-time.Hour),
	}
	usecase := NewCheckoutService(repo, repo)

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
		RightStatus: domain.PurchaseRightStatusUsed,
		expiresAt:   time.Now().Add(time.Hour),
	}
	usecase := NewCheckoutService(repo, repo)

	_, _, allowed, _, err := usecase.CheckAccess(context.Background(), uuid.New(), uuid.New())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected allowed=false for RightStatus=used")
	}
}

// Права не существовало
func TestCheckAccess_NoRows(t *testing.T) {
	repo := mockRepo{
		findErr: pgx.ErrNoRows,
	}
	usecase := NewCheckoutService(repo, repo)

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

// Ошибка БД
func TestCheckAccess_DBError(t *testing.T) {
	repo := mockRepo{
		findErr: errors.New("connection refused"),
	}
	usecase := NewCheckoutService(repo, repo)

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
	usecase := NewCheckoutService(repo, repo)

	success, _, err := usecase.Pay(context.Background(), uuid.New(), uuid.New())

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
	usecase := NewCheckoutService(repo, repo)

	success, reason, err := usecase.Pay(context.Background(), uuid.New(), uuid.New())

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
	usecase := NewCheckoutService(repo, repo)

	success, _, err := usecase.Pay(context.Background(), uuid.New(), uuid.New())

	if err == nil {
		t.Fatal("expected non-nil error for db failure")
	}
	if success {
		t.Error("expected success=false on db error")
	}
}
