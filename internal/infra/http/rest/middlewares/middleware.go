package middlewares

import (
	"avito-queue/internal/infra/http/rest/handlers"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func PurchaseRightMiddleware(usecase handlers.CheckAccessor) gin.HandlerFunc {
	return func(c *gin.Context) {
		// userID передаётся заголовком X-User-ID, itemID — частью пути (/checkout/:itemID)
		rawUserID := c.GetString("userID")
		userUUID, err := uuid.Parse(rawUserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные пользователя"})
			return
		}

		if userUUID == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Пользователь не определен"})
			return
		}

		itemID := c.Param("itemID")
		itemUUID, err := uuid.Parse(itemID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные товара"})
			return
		}

		purchaseID, expiresAt, allowed, reason, err := usecase.CheckAccess(c.Request.Context(), userUUID, itemUUID)
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
