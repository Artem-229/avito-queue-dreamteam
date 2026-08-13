package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"avito-queue/internal/infra/http/rest/middlewares"
)

type AdminLoginRequest struct {
	SecretKey string `json:"secret_key" binding:"required"`
}

// AdminLogin — не создаёт сессию/токен, только подтверждает верность ключа
// (через middlewares.ValidateAdminKey — ту же функцию, что использует
// AdminAuthMiddleware, чтобы проверка не разошлась между двумя местами).
// Фронт после успешного ответа сохраняет введённый ключ и дальше шлёт его
// заголовком X-Admin-Key на каждый защищённый запрос — тот же ключ и есть
// "сессия".
func (h *Handlers) AdminLogin(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req AdminLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "bad_request", "message": "Некорректный запрос"})
			return
		}

		if !middlewares.ValidateAdminKey(req.SecretKey, secretKey) {
			c.JSON(http.StatusForbidden, gin.H{"code": "admin_forbidden", "message": "Неверный ключ"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}
