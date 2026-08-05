package checkout

import (
	"github.com/gin-gonic/gin"
)

type CheckoutHandler struct {
	Repo *PurchaseRightRepo
}

func (h CheckoutHandler) Checkout() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func (h CheckoutHandler) Pay() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}
