package rest

import (
	"avito-queue/internal/infra/http/rest/handlers"
	"avito-queue/internal/infra/http/rest/middlewares"
	"log/slog"

	"github.com/gin-gonic/gin"
)

func NewRouter(h *handlers.Handlers, logger *slog.Logger) *gin.Engine {
	router := gin.Default()

	router.GET("/health", h.Health)

	api := router.Group("/api/v1")
	api.Use(middlewares.TestAuthMiddleware())
	api.Use(middlewares.LoggerMiddleware(logger))
	{
		api.GET("/catalog", h.Catalog.GetList)
		api.GET("/catalog/:id", h.Catalog.GetByID)
		api.POST("/catalog/:id/buy", h.Catalog.BuyItem)
		api.POST("/catalog/:id/queue", h.Queue.Join)
	}

	return router
}
