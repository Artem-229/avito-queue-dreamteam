package middlewares

import (
	"github.com/gin-gonic/gin"
)

func TestAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-Test-User-Id")

		if userID == "" {
			userID = "00000000-0000-0000-0000-000000000000"
		}

		c.Set("userID", userID)
		c.Next()
	}
}
