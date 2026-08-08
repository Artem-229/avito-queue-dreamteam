package handlers

import (
	"avito-queue/internal/domain"
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CatalogService interface {
	GetCatalog(ctx context.Context) ([]domain.CatalogItem, error)
	GetCatalogItem(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error)
	GetSimilarItems(ctx context.Context, id uuid.UUID) ([]domain.CatalogItem, error)
}

type PurchaseRightService interface {
	Buy(ctx context.Context, userID, itemID uuid.UUID) error
}

type CatalogHandler struct {
	catalogService       CatalogService
	purchaseRightService PurchaseRightService
}

func NewCatalogHandler(s CatalogService, p PurchaseRightService) *CatalogHandler {
	return &CatalogHandler{
		catalogService:       s,
		purchaseRightService: p,
	}
}

func (h *CatalogHandler) GetList(c *gin.Context) {
	items, err := h.catalogService.GetCatalog(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch catalog"})
		return
	}

	if items == nil {
		items = []domain.CatalogItem{}
	}
	c.JSON(http.StatusOK, items)
}

func (h *CatalogHandler) GetByID(c *gin.Context) {
	rawID := c.Param("id")

	id, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse id"})
		return
	}

	item, err := h.catalogService.GetCatalogItem(c.Request.Context(), id)
	if err != nil {
		c.JSON(mapErrorIntoStatusCodes(err), gin.H{"error": "item not found"})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *CatalogHandler) BuyItem(c *gin.Context) {
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse id"})
		return
	}

	userID, err := userIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if err := h.purchaseRightService.Buy(c.Request.Context(), userID, itemID); err != nil {
		switch {
		case errors.Is(err, domain.ErrNoPurchaseRight):
			c.JSON(http.StatusForbidden, gin.H{"error": "no granted purchase right for this item"})
		case errors.Is(err, domain.ErrItemSoldOut):
			c.JSON(http.StatusConflict, gin.H{"error": "item is sold out"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to buy item"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "purchase completed",
		"item_id": itemID,
	})
}

func (h *CatalogHandler) GetSimilarItems(c *gin.Context) {
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse id"})
		return
	}

	items, err := h.catalogService.GetSimilarItems(c.Request.Context(), itemID)
	if err != nil {
		c.JSON(mapErrorIntoStatusCodes(err), gin.H{"error": "failed to fetch similar items"})
		return
	}

	if items == nil {
		items = []domain.CatalogItem{}
	}

	c.JSON(http.StatusOK, items)
}
