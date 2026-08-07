package handlers

import (
	"avito-queue/internal/domain"
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type QueueService interface {
	Entry(ctx context.Context, userID, itemID uuid.UUID) error
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
