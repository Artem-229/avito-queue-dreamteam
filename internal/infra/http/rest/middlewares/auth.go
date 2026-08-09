package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func SessionAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawUserID string

		cookie, err := c.Cookie("session")
		if err == nil {
			parts := strings.Split(cookie, ".")
			if len(parts) == 2 {
				id := parts[0]
				sig := parts[1]

				if Verify(id, sig, secret) {
					rawUserID = id
				}
			}
		}

		if rawUserID == "" {
			rawUserID = c.GetHeader("X-User-Id")
		}

		if rawUserID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "требуется авторизация"})
			c.Abort()
			return
		}

		userID, err := uuid.Parse(rawUserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "невалидный формат user_id"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}
