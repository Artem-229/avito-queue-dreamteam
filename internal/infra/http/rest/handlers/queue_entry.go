package handlers

import (
	"avito-queue/internal/domain"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type QueueEntryService interface {
	Entry(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueStatusResponse, error)
	Status(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueStatusResponse, error)
	Leave(ctx context.Context, userID, itemID uuid.UUID) (domain.QueueStatusResponse, error)
	Buy(ctx context.Context, userID, itemID uuid.UUID) error
}

type QueueEntryHandler struct {
	queueEntryService QueueEntryService
}

func NewQueueEntryHandler(q QueueEntryService) *QueueEntryHandler {
	return &QueueEntryHandler{
		queueEntryService: q,
	}
}

// Join встаёт в очередь и возвращает тот же конверт статуса, что и GetStatus —
// фронту не нужен отдельный запрос сразу после входа.
func (h *QueueEntryHandler) Join(c *gin.Context) {
	itemID, ok := itemIDFromParam(c)
	if !ok {
		return
	}

	userID, err := userIDFromContext(c)
	if err != nil {
		respondErrorCode(c, http.StatusUnauthorized, CodeUnauthorized, "Требуется авторизация")
		return
	}

	response, err := h.queueEntryService.Entry(c.Request.Context(), userID, itemID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *QueueEntryHandler) GetStatus(c *gin.Context) {
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
	response, err := h.queueEntryService.Status(c.Request.Context(), userID, itemID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *QueueEntryHandler) Leave(c *gin.Context) {
	itemID, ok := itemIDFromParam(c)
	if !ok {
		return
	}

	userID, err := userIDFromContext(c)
	if err != nil {
		respondErrorCode(c, http.StatusUnauthorized, CodeUnauthorized, "Требуется авторизация")
		return
	}

	response, err := h.queueEntryService.Leave(c.Request.Context(), userID, itemID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// Purchase — единственный путь покупки. Проверка права целиком внутри
// PurchaseRight.Buy: отдельный читающий гейт перед покупкой не нужен, гарантию
// даёт только транзакция самой покупки.
func (h *QueueEntryHandler) Purchase(c *gin.Context) {
	itemID, ok := itemIDFromParam(c)
	if !ok {
		return
	}

	userID, err := userIDFromContext(c)
	if err != nil {
		respondErrorCode(c, http.StatusUnauthorized, CodeUnauthorized, "Требуется авторизация")
		return
	}

	if err := h.queueEntryService.Buy(c.Request.Context(), userID, itemID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Покупка завершена",
		"item_id": itemID,
	})
}
