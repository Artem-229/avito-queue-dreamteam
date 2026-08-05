package domain

import "github.com/google/uuid"

type PurchaseRightStatus string

const (
	PurchaseRightStatusGranted   PurchaseRightStatus = "granted"
	PurchaseRightStatusUsed      PurchaseRightStatus = "used"
	PurchaseRightStatusCancelled PurchaseRightStatus = "cancelled"
	PurchaseRightStatusExpired   PurchaseRightStatus = "expired"
)

type PurchaseRight struct {
	ID     uuid.UUID
	UserID uuid.UUID
	ItemID uuid.UUID
	Status PurchaseRightStatus
}

func (p *PurchaseRight) IsGranted() bool {
	return p.Status == PurchaseRightStatusGranted
}
