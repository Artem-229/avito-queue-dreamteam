package checkout

import "github.com/gin-gonic/gin"

func RegisterCheckoutRoutes(r *gin.Engine, h *CheckoutHandler) {
	checkout := r.Group("/checkout/:itemID")
	checkout.Use(h.PurchaseRightMiddleware())
	checkout.GET("", h.Checkout())
	checkout.POST("/pay", h.Pay())
}
