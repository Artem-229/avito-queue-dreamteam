package domain

import (
	"time"

	"github.com/google/uuid"
)

type PurchaseStatus string

const (
	StatusGranted   PurchaseStatus = "granted"
	StatusUsed      PurchaseStatus = "used"
	StatusCancelled PurchaseStatus = "cancelled"
	StatusExpired   PurchaseStatus = "expired"
)

type PurchaseRight struct {
	ID        uuid.UUID      `json:"id"`
	ItemID    uuid.UUID      `json:"item_id"`
	UserID    uuid.UUID      `json:"user_id"`
	Status    PurchaseStatus `json:"status"`
	ExpiresAt time.Time      `json:"expires_at"`
	CreatedAt time.Time      `json:"created_at"`
}
