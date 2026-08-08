package handlers

import (
	"avito-queue/internal/domain"
	"errors"
)

type Handlers struct {
	Catalog  *CatalogHandler
	Checkout *CheckoutHandler
	Queue    *QueueHandler
}

func New(catalog *CatalogHandler, queue *QueueHandler, checkout *CheckoutHandler) *Handlers {
	return &Handlers{
		Catalog:  catalog,
		Checkout: checkout,
		Queue:    queue,
	}
}

func mapErrorIntoStatusCodes(err error) int {
	switch {
	case errors.Is(err, domain.ErrNoItemFound):
		return 404
	case errors.Is(err, domain.ErrNoPurchaseRight):
		return 403
	default:
		return 500
	}
}
