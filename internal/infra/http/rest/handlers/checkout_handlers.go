package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CheckAccessor interface {
	CheckAccess(ctx context.Context, userID, itemID uuid.UUID) (purchaseID uuid.UUID, expiresAt time.Time, allowed bool, reason string, err error)
	Pay(ctx context.Context, purchaseID uuid.UUID) (success bool, reason string, err error)
}

type CheckoutHandler struct {
	Usecase CheckAccessor
}

func (h CheckoutHandler) GetStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		purchaseID := c.MustGet("purchase_id").(uuid.UUID)
		expiresAt := c.MustGet("expires_at").(time.Time)

		c.JSON(http.StatusOK, gin.H{
			"purchase_id": purchaseID,
			"expires_at":  expiresAt,
		})
	}
}

func (h CheckoutHandler) Pay() gin.HandlerFunc {
	return func(c *gin.Context) {
		purchaseID := c.MustGet("purchase_id").(uuid.UUID)
		success, reason, err := h.Usecase.Pay(c.Request.Context(), purchaseID)

		switch {
		case err != nil:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
			return
		case !success:
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": reason})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": "Покупка завершена"})
	}
}
