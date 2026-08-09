package handlers

import (
	"avito-queue/internal/domain"
	"errors"
	"net/http"
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

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func mapErrorIntoStatusCodes(err error) (int, ErrorResponse) {
	switch {
	case errors.Is(err, domain.ErrNoItemFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    "ITEM_NOT_FOUND",
			Message: "Товар не найден",
		}
	case errors.Is(err, domain.ErrNoPurchaseRight):
		return http.StatusForbidden, ErrorResponse{
			Code:    "NO_PURCHASE_RIGHT",
			Message: "У вас нет активного права на покупку этого товара",
		}
	case errors.Is(err, domain.ErrItemSoldOut):
		return http.StatusConflict, ErrorResponse{
			Code:    "ITEM_SOLD_OUT",
			Message: "К сожалению, товар закончился",
		}
	case errors.Is(err, domain.ErrUserAlreadyInQueue):
		return http.StatusConflict, ErrorResponse{
			Code:    "ALREADY_IN_QUEUE",
			Message: "Вы уже находитесь в очереди за этим товаром",
		}
	case errors.Is(err, domain.ErrUserNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    "NOT_IN_QUEUE",
			Message: "Вы не находитесь в очереди за этим товаром",
		}
	default:
		return http.StatusInternalServerError, ErrorResponse{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Внутренняя ошибка сервера",
		}
	}
}
