package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"avito-queue/internal/domain"
)

type QueueService interface {
	Entry(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueStatusResponse, error)
	GetStatus(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueStatusResponse, error)
	Leave(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueStatusResponse, error)
}

// PurchaseRightService — трата выданного права. Живёт в хендлере очереди, а не
// каталога: покупка — финальное состояние сценария очереди, а не операция над
// карточкой товара.
type PurchaseRightService interface {
	Buy(ctx context.Context, userID, itemID uuid.UUID) error
}

type QueueHandler struct {
	queueService         QueueService
	purchaseRightService PurchaseRightService
}

func NewQueueHandler(s QueueService, p PurchaseRightService) *QueueHandler {
	return &QueueHandler{
		queueService:         s,
		purchaseRightService: p,
	}
}

// Join встаёт в очередь и возвращает тот же конверт статуса, что и GetStatus —
// фронту не нужен отдельный запрос сразу после входа.
func (h *QueueHandler) Join(c *gin.Context) {
	itemID, ok := itemIDFromParam(c)
	if !ok {
		return
	}

	userID, err := userIDFromContext(c)
	if err != nil {
		respondErrorCode(c, http.StatusUnauthorized, CodeUnauthorized, "Требуется авторизация")
		return
	}

	response, err := h.queueService.Entry(c.Request.Context(), userID, itemID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *QueueHandler) GetStatus(c *gin.Context) {
	itemID, ok := itemIDFromParam(c)
	if !ok {
		return
	}

	userID, err := userIDFromContext(c)
	if err != nil {
		respondErrorCode(c, http.StatusUnauthorized, CodeUnauthorized, "Требуется авторизация")
		return
	}

	// «Не в очереди» — не ошибка, а состояние not_in_queue в том же конверте,
	// поэтому 404 здесь не возвращается.
	response, err := h.queueService.GetStatus(c.Request.Context(), userID, itemID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *QueueHandler) Leave(c *gin.Context) {
	itemID, ok := itemIDFromParam(c)
	if !ok {
		return
	}

	userID, err := userIDFromContext(c)
	if err != nil {
		respondErrorCode(c, http.StatusUnauthorized, CodeUnauthorized, "Требуется авторизация")
		return
	}

	response, err := h.queueService.Leave(c.Request.Context(), userID, itemID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// Purchase — единственный путь покупки. Проверка права целиком внутри
// PurchaseRight.Buy: отдельный читающий гейт перед покупкой не нужен, гарантию
// даёт только транзакция самой покупки.
func (h *QueueHandler) Purchase(c *gin.Context) {
	itemID, ok := itemIDFromParam(c)
	if !ok {
		return
	}

	userID, err := userIDFromContext(c)
	if err != nil {
		respondErrorCode(c, http.StatusUnauthorized, CodeUnauthorized, "Требуется авторизация")
		return
	}

	if err := h.purchaseRightService.Buy(c.Request.Context(), userID, itemID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Покупка завершена",
		"item_id": itemID,
	})
}
