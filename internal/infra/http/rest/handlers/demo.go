package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DemoService interface {
	ExpireNow(ctx context.Context, itemID uuid.UUID) error
	Simulate(ctx context.Context, itemID uuid.UUID, count int) (joined, failed int, err error)
}

type DemoHandler struct {
	demoService DemoService
}

func NewDemoHandler(s DemoService) *DemoHandler {
	return &DemoHandler{demoService: s}
}

func (h *DemoHandler) ExpireNow(c *gin.Context) {
	itemID, ok := itemIDFromParam(c)
	if !ok {
		return
	}

	if err := h.demoService.ExpireNow(c.Request.Context(), itemID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Права на покупку сожжены", "item_id": itemID})
}

// simulateRequest — потолок count стоит здесь, а не только на фронте: число
// одновременных Entry ограничено размером пула соединений, а интерфейс можно
// обойти одним curl.
type simulateRequest struct {
	ItemID uuid.UUID `json:"item_id" binding:"required"`
	Count  int       `json:"count" binding:"required,min=1,max=100"`
}

func (h *DemoHandler) Simulate(c *gin.Context) {
	var req simulateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorCode(c, http.StatusBadRequest, CodeInvalidBody,
			"Укажите товар и число покупателей от 1 до 100")
		return
	}

	joined, failed, err := h.demoService.Simulate(c.Request.Context(), req.ItemID, req.Count)
	if err != nil {
		respondErrorCode(c, http.StatusInternalServerError, CodeSimulateFailure, "Не удалось запустить симуляцию")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item_id":   req.ItemID,
		"requested": req.Count,
		"joined":    joined,
		"failed":    failed,
	})
}
