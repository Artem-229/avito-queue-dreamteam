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
	ID        uuid.UUID
	UserID    uuid.UUID
	ItemID    uuid.UUID
	Status    PurchaseRightStatus
	CreatedAt *time.Time
	ExpiredAt *time.Time
}

func (p *PurchaseRight) IsGranted() bool {
	return p.Status == PurchaseRightStatusGranted
}
