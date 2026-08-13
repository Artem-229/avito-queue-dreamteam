package domain

import (
	"time"

	"github.com/google/uuid"
)

type CatalogItem struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	PriceKopecks   int64      `json:"price_kopecks"`
	TotalStock     int        `json:"total_stock"`
	HoldTTLSeconds int        `json:"hold_ttl_seconds"`
	GrantedCount   int        `json:"granted_count"`
	UsedCount      int        `json:"used_count"`
	Category       string     `json:"category"`
	SellerName     string     `json:"seller_name"`
	CreatedAt      time.Time  `json:"created_at"`
	DeletedAt      *time.Time `json:"-"`

	Embedding []float32 `json:"-"`
}

// CatalogItemCard — товар плюс то, что нужно только карточке: сколько человек
// уже ждёт и какой шанс у того, кто встанет сейчас.
type CatalogItemCard struct {
	CatalogItem
	QueueSize    int             `json:"queue_size"`
	ChanceIfJoin *PurchaseChance `json:"chance_if_join"`
}
