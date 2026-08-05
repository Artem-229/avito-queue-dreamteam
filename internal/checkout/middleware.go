package checkout

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func (h CheckoutHandler) PurchaseRightMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		//условимся, что userID передает фронт и каталог передает их через Query-параметр, а itemID считается из строки
		userID := c.Query("userID")
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

		idPurchase, status, expiresAt, err := h.Repo.FindByUserAndItem(c.Request.Context(), intUserID, intItemID)

		switch {
		case errors.Is(err, pgx.ErrNoRows):
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Нет права на покупку этого товара"})
			return
		case err != nil:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
			return
		case status != "granted" || !expiresAt.After(time.Now()):
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Право на покупку товара неактивно"})
			return
		}

		c.Set("purchase_id", idPurchase)
		c.Next()
	}
}
