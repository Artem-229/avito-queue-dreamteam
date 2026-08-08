package domain

import (
	"time"

	"github.com/google/uuid"
)

type QueueStatus string

const (
	QueueStatusWaiting   QueueStatus = "waiting"
	QueueStatusCancelled QueueStatus = "cancelled"
	QueueStatusGranted   QueueStatus = "granted"
	QueueStatusSoldOut   QueueStatus = "sold_out"
	QueueStatusExpired   QueueStatus = "expired"
	QueueStatusPurchased QueueStatus = "purchased"
)

type Queue struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ItemID    uuid.UUID
	Status    QueueStatus
	CreatedAt time.Time
	DeletedAt *time.Time
}
