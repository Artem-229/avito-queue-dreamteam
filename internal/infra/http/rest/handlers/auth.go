package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"avito-queue/internal/infra/http/rest/session"
)

type DemoLoginRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
}

// DemoLogin выдаёт личность тестовому пользователю: настоящей регистрации в
// MVP нет, но право на покупку обязано к кому-то крепиться.
//
// user_id генерируется случайно, а не выводится из display_name — иначе вход
// под чужим именем забирал бы чужую позицию в очереди. Имя нигде не хранится,
// это только подпись под аватаркой.
//
// Токен отдаётся кукой и в теле ответа
func (h *Handlers) DemoLogin(secret string, secureCookie bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DemoLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondErrorCode(c, http.StatusBadRequest, CodeInvalidBody, "Укажите имя тестового пользователя")
			return
		}

		userID := uuid.New()
		token := session.Issue(userID, secret, time.Now())

		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(session.CookieName, token, int(session.TTL.Seconds()), "/", "", secureCookie, true)

		c.JSON(http.StatusOK, gin.H{
			"user_id":      userID,
			"display_name": req.DisplayName,
			"token":        token,
		})
	}
}
