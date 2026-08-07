package domain

import (
	"time"

	"github.com/google/uuid"
)

type PurchaseRightStatus string

const (
	PurchaseRightStatusGranted   PurchaseRightStatus = "granted"
	PurchaseRightStatusUsed      PurchaseRightStatus = "used"
	PurchaseRightStatusCancelled PurchaseRightStatus = "cancelled"
	PurchaseRightStatusExpired   PurchaseRightStatus = "expired"
)

type PurchaseRight struct {
	ID        uuid.UUID           `json:"id"`
	ItemID    uuid.UUID           `json:"item_id"`
	UserID    uuid.UUID           `json:"user_id"`
	Status    PurchaseRightStatus `json:"status"`
	ExpiresAt time.Time           `json:"expires_at"`
	CreatedAt time.Time           `json:"created_at"`
}
