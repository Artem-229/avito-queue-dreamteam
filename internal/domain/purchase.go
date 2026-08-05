package entity

import "time"

type PurchaseStatus string

const (
	StatusGranted   PurchaseStatus = "granted"
	StatusUsed      PurchaseStatus = "used"
	StatusCancelled PurchaseStatus = "cancelled"
	StatusExpired   PurchaseStatus = "expired"
)

type PurchaseRight struct {
	ID        string         `json:"id"`
	ItemID    string         `json:"item_id"`
	UserID    string         `json:"user_id"`
	Status    PurchaseStatus `json:"status"`
	ExpiredAt time.Time      `json:"expires_at"`
	CreatedAt time.Time      `json:"created_at"`
}
