package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"avito-queue/internal/infra/http/rest/middlewares"
)

// Коды ошибок конкретно для admin-логина — рядом с остальными Code* в
// handler.go по смыслу, но объявлены здесь, чтобы не разрастать общий файл
// сущностями, специфичными только для одной ручки.
const (
	CodeAdminForbidden = "ADMIN_FORBIDDEN"
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
			respondErrorCode(c, http.StatusBadRequest, CodeInvalidBody, "Некорректный запрос")
			return
		}

		if !middlewares.ValidateAdminKey(req.SecretKey, secretKey) {
			respondErrorCode(c, http.StatusForbidden, CodeAdminForbidden, "Неверный ключ")
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}
