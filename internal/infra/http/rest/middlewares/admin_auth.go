package middlewares

import (
	"crypto/hmac"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ValidateAdminKey — единственное место, где сравнивается ключ администратора.
// Используется и здесь (middleware), и в handlers.AdminLogin — чтобы логика
// проверки не разошлась между двумя местами при будущих правках.
func ValidateAdminKey(provided, secretKey string) bool {
	if secretKey == "" || provided == "" {
		return false
	}
	return hmac.Equal([]byte(provided), []byte(secretKey))
}

// AdminAuthMiddleware — проверка секретного ключа администратора в заголовке.
// Ключ сам по себе служит "пропуском" на каждый запрос — отдельная сессия
// не нужна для внутреннего инструмента команды.
func AdminAuthMiddleware(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secretKey == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, errorBody{
				Code:    "ADMIN_DISABLED",
				Message: "Административные функции не настроены на этом сервере",
			})
			return
		}

		if !ValidateAdminKey(c.GetHeader("X-Admin-Key"), secretKey) {
			c.AbortWithStatusJSON(http.StatusForbidden, errorBody{
				Code:    "ADMIN_FORBIDDEN",
				Message: "Неверный или отсутствующий ключ администратора",
			})
			return
		}

		c.Next()
	}
}
