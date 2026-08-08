package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ItemIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		itemUUID, err := uuid.Parse(c.Param("itemID"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные товара"})
			return
		}

		c.Set("itemID", itemUUID)
		c.Next()
	}
}
