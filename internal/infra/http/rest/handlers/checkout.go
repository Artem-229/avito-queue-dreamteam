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
	Pay(ctx context.Context, userID, itemID uuid.UUID) (success bool, reason string, err error)
}

type CheckoutHandler struct {
	Service CheckAccessor
}

func NewCheckoutHandler(service CheckAccessor) *CheckoutHandler {
	return &CheckoutHandler{Service: service}
}

func (h *CheckoutHandler) GetStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("userID").(uuid.UUID)
		itemID := c.MustGet("itemID").(uuid.UUID)

		purchaseID, expiresAt, allowed, reason, err := h.Service.CheckAccess(c, userID, itemID)
		if err != nil {
			c.JSON(mapErrorIntoStatusCodes(err), gin.H{"error": "не удалось проверить право на покупку"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"purchase_id": purchaseID,
			"allowed":     allowed,
			"expires_at":  expiresAt,
			"reason":      reason,
		})
	}
}

func (h *CheckoutHandler) Pay() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("userID").(uuid.UUID)
		itemID := c.MustGet("itemID").(uuid.UUID)

		success, reason, err := h.Service.Pay(c.Request.Context(), userID, itemID)

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
