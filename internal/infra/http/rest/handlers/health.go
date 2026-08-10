package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Pinger — проверка живости хранилища. Health без него отвечает «ok» с мёртвым
// Postgres: оркестратор и docker healthcheck считают контейнер здоровым, пока
// пользователи получают 500 на каждом запросе.
type Pinger interface {
	Ping(ctx context.Context) error
}

const healthCheckTimeout = 2 * time.Second

func (h *Handlers) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), healthCheckTimeout)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "degraded",
			"db":     "unreachable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"db":     "ok",
	})
}
