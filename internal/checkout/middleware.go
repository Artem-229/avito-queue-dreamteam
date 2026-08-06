package checkout

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h CheckoutHandler) PurchaseRightMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// userID передаётся заголовком X-User-ID, itemID — частью пути (/checkout/:itemID)
		userID := c.GetHeader("X-User-ID")
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные пользователя"})
			return
		}

		itemID := c.Param("itemID")
		itemUUID, err := uuid.Parse(itemID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные товара"})
			return
		}

		purchaseID, expiresAt, allowed, reason, err := h.Usecase.CheckAccess(c.Request.Context(), userUUID, itemUUID)
		switch {
		case err != nil:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
			return
		case !allowed:
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": reason})
			return
		}

		c.Set("purchase_id", purchaseID)
		c.Set("expires_at", expiresAt)
		c.Next()
	}
}
