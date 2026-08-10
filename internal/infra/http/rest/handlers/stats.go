package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"avito-queue/internal/domain"
)

type StatsService interface {
	GetStats(ctx context.Context) (domain.QueueStats, error)
}

type StatsHandler struct {
	statsService StatsService
}

func NewStatsHandler(s StatsService) *StatsHandler {
	return &StatsHandler{statsService: s}
}

// GetStats — витрина метрик очереди (GET /api/v1/stats).
func (h *StatsHandler) GetStats(c *gin.Context) {
	stats, err := h.statsService.GetStats(c.Request.Context())
	if err != nil {
		respondErrorCode(c, http.StatusInternalServerError, CodeStatsFailure, "Не удалось собрать метрики")
		return
	}

	c.JSON(http.StatusOK, stats)
}
