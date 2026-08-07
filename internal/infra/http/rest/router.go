package rest

import (
	"avito-queue/internal/infra/http/rest/handlers"
	"avito-queue/internal/infra/http/rest/middlewares"

	"github.com/gin-gonic/gin"
)

func NewRouter(h *handlers.Handlers) *gin.Engine {
	router := gin.Default()

	router.GET("/health", h.Health)

	api := router.Group("/api/v1")
	api.Use(middlewares.TestAuthMiddleware())
	{
		api.GET("/catalog", h.Catalog.GetList)
		api.GET("/catalog/:id", h.Catalog.GetByID)
		api.POST("/catalog/:id/buy", h.Catalog.BuyItem)
	}

	checkout := router.Group("/checkout/:itemID")
	checkout.Use(middlewares.TestAuthMiddleware())
	checkout.Use(middlewares.PurchaseRightMiddleware(h.Checkout.Usecase))
	checkout.GET("", h.Checkout.GetStatus())
	checkout.POST("/pay", h.Checkout.Pay())
	return router
}
