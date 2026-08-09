package handlers

import (
	"avito-queue/internal/domain"
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type QueueService interface {
	Entry(ctx context.Context, userID, itemID uuid.UUID) error
	GetStatus(ctx context.Context, userID, itemID uuid.UUID) (status domain.QueueStatus, position int, expiresAt *time.Time, err error)
}

type QueueHandler struct {
	queueService QueueService
}

func NewQueueHandler(s QueueService) *QueueHandler {
	return &QueueHandler{
		queueService: s,
	}
}

func (h *QueueHandler) Join(c *gin.Context) {
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse id"})
		return
	}

	userID, err := userIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if err := h.queueService.Entry(c.Request.Context(), userID, itemID); err != nil {
		if errors.Is(err, domain.ErrUserAlreadyInQueue) {
			c.JSON(http.StatusConflict, gin.H{"error": "already in queue"})
			return
		}
		if errors.Is(err, domain.ErrNoItemFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join queue"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "joined queue", "item_id": itemID})
}

func (h *QueueHandler) GetStatus(c *gin.Context) {
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse id"})
		return
	}

	userID, err := userIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	status, position, expiresAt, err := h.queueService.GetStatus(c.Request.Context(), userID, itemID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "you are not in queue for this item"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get status"})
		return
	}

	response := domain.QueueStatusResponse{
		Status:    status,
		Position:  position,
		ExpiresAt: expiresAt,
	}

	switch status {
	case domain.QueueStatusWaiting:
		response.Message = "Вы в очереди. Пожалуйста, подождите освобождения слота."
	case domain.QueueStatusGranted:
		response.Message = "Ваша очередь подошла! У вас есть несколько минут на оплату."
	case domain.QueueStatusPurchased:
		response.Message = "Покупка успешно завершена."
	case domain.QueueStatusExpired:
		response.Message = "Время вышло, право на покупку сгорело. Вы можете встать в очередь заново."
	case domain.QueueStatusSoldOut:
		response.Message = "К сожалению, товар закончился. Посмотрите похожие лоты."
	case domain.QueueStatusCancelled:
		response.Message = "Вы покинули очередь."
	}

	c.JSON(http.StatusOK, response)
}
