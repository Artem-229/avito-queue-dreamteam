package domain

import (
	"time"

	"github.com/google/uuid"
)

type CatalogItem struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Price      float64    `json:"price"`
	TotalStock int        `json:"total_stock"`
	CreatedAt  time.Time  `json:"created_at"`
	DeletedAt  *time.Time `json:"-"`
}
