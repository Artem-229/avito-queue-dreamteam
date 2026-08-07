package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func userIDFromContext(c *gin.Context) (uuid.UUID, error) {
	raw, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, errors.New("userID missing from context")
	}

	userID, ok := raw.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("userID in context has unexpected type")
	}

	return userID, nil
}
