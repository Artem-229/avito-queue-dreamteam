package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"avito-queue/internal/domain"
)

type Handlers struct {
	Catalog *CatalogHandler
	Queue   *QueueHandler
	Demo    *DemoHandler
	Stats   *StatsHandler
	db      Pinger
}

func New(
	catalog *CatalogHandler,
	queue *QueueHandler,
	demo *DemoHandler,
	stats *StatsHandler,
	db Pinger,
) *Handlers {
	return &Handlers{
		Catalog: catalog,
		Queue:   queue,
		Demo:    demo,
		Stats:   stats,
		db:      db,
	}
}

// ErrorResponse — единый формат тела ошибки на всё API: code — стабильная
// строка для клиента, message — текст для человека.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Коды ошибок, не имеющие доменного сентинела: их источник — сам HTTP-слой.
const (
	CodeInvalidItemID   = "INVALID_ITEM_ID"
	CodeRouteNotFound   = "ROUTE_NOT_FOUND"
	CodeMethodNotAllwd  = "METHOD_NOT_ALLOWED"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeInvalidBody     = "INVALID_REQUEST_BODY"
	CodeDemoDisabled    = "DEMO_DISABLED"
	CodeInternal        = "INTERNAL_SERVER_ERROR"
	CodeCatalogFailure  = "CATALOG_UNAVAILABLE"
	CodeStatsFailure    = "STATS_UNAVAILABLE"
	CodeSimulateFailure = "SIMULATION_FAILED"
)

// NotFound и MethodNotAllowed отвечают за роутер: единый конверт ошибки нужен
// и на путях, до хендлеров не дошедших.
func (h *Handlers) NotFound(c *gin.Context) {
	respondErrorCode(c, http.StatusNotFound, CodeRouteNotFound, "Такого метода API не существует")
}

func (h *Handlers) MethodNotAllowed(c *gin.Context) {
	respondErrorCode(c, http.StatusMethodNotAllowed, CodeMethodNotAllwd, "Метод не поддерживается этим адресом")
}

func respondError(c *gin.Context, err error) {
	status, body := mapErrorIntoStatusCodes(err)
	c.JSON(status, body)
}

func respondErrorCode(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{Code: code, Message: message})
}

// itemIDFromParam разбирает :id и сам отвечает клиенту при ошибке.
func itemIDFromParam(c *gin.Context) (uuid.UUID, bool) {
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondErrorCode(c, http.StatusBadRequest, CodeInvalidItemID, "Некорректный идентификатор товара")
		return uuid.Nil, false
	}

	return itemID, true
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
	case errors.Is(err, domain.ErrCannotLeaveQueue):
		return http.StatusConflict, ErrorResponse{
			Code:    "CANNOT_LEAVE_QUEUE",
			Message: "Из текущего состояния выйти из очереди нельзя",
		}
	case errors.Is(err, domain.ErrStatusConflict):
		return http.StatusConflict, ErrorResponse{
			Code:    "STATUS_CONFLICT",
			Message: "Состояние изменилось, обновите страницу",
		}
	default:
		return http.StatusInternalServerError, ErrorResponse{
			Code:    CodeInternal,
			Message: "Внутренняя ошибка сервера",
		}
	}
}
