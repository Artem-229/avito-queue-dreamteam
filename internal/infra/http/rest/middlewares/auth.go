package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("X-User-Id")
		if raw == "" {
			raw = "00000000-0000-0000-0000-000000000000"
		}

		userID, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid X-User-Id header"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}
