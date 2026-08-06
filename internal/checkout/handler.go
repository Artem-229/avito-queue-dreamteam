package checkout

import (
	"context"

	"github.com/gin-gonic/gin"
)

type CheckAccessor interface {
	CheckAccess(ctx context.Context, userID, itemID int) (purchaseID int, allowed bool, reason string, err error)
}

type CheckoutHandler struct {
	Usecase CheckAccessor
}

func (h CheckoutHandler) Checkout() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func (h CheckoutHandler) Pay() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}
