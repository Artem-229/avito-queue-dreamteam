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

	// QueueStatusNotInQueue — виртуальное состояние, существует только в
	// конверте ответа: в таблице queue его нет (INV-8).
	QueueStatusNotInQueue QueueStatus = "not_in_queue"
)

type Queue struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ItemID    uuid.UUID
	Status    QueueStatus
	CreatedAt time.Time
}

// NextStep — подсказка для UI, что делать дальше в текущем состоянии (INV-8).
type NextStep struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

type QueueStatusResponse struct {
	Status    QueueStatus `json:"status"`
	Message   string      `json:"message"`
	Position  int         `json:"position,omitempty"`
	ExpiresAt *time.Time  `json:"expires_at,omitempty"`
	// QueueSize — сколько человек сейчас в статусе waiting по этому товару.
	QueueSize int `json:"queue_size"`
	// ETASeconds сознательно nil: формула по позиции без данных о реальной
	// конверсии права в покупку легко была бы неверной.
	ETASeconds *int `json:"eta_seconds"`
	// ServerTime — время Postgres, не Go (INV-6).
	ServerTime time.Time `json:"server_time"`
	NextStep   NextStep  `json:"next_step"`
	// Alternatives — похожие лоты для sold_out/expired, иначе пустой срез
	// (не nil, чтобы в JSON было [], а не null).
	Alternatives []uuid.UUID `json:"alternatives"`
}
