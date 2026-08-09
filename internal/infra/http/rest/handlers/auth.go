package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"avito-queue/internal/infra/http/rest/middlewares"
)

type DemoLoginRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
}

func (h *Handlers) DemoLogin(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DemoLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректное тело запроса"})
			return
		}

		userID := uuid.New().String()

		signature := middlewares.Sign(userID, secret)

		cookieValue := userID + "." + signature

		c.SetCookie("session", cookieValue, int(24*time.Hour/time.Second), "/", "", false, true)

		c.JSON(http.StatusOK, gin.H{
			"user_id":      userID,
			"display_name": req.DisplayName,
			"message":      "Успешный вход",
		})
	}
}
