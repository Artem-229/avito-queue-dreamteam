package checkout

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h CheckoutHandler) PurchaseRightMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		//условимся, что itemID передает фронт и каталог передает их через Query-параметр, а userID считается из заголовка
		userID := c.GetHeader("X-User-ID")
		intUserID, err := strconv.Atoi(userID)
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": "Некорректные данные пользователя"})
			return
		}

		itemID := c.Param("itemID")
		intItemID, err := strconv.Atoi(itemID)
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": "Некорректные данные товара"})
			return
		}

		purchaseID, allowed, reason, err := h.Usecase.CheckAccess(c.Request.Context(), intUserID, intItemID)
		switch {
		case err != nil:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
			return
		case !allowed:
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": reason})
			return
		}

		c.Set("purchase_id", purchaseID)
		c.Next()
	}
}
