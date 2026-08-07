package handlers

import (
	"avito-queue/internal/domain"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CatalogService interface {
	GetCatalog(ctx context.Context) ([]domain.CatalogItem, error)
	GetCatalogItem(ctx context.Context, id string) (domain.CatalogItem, error)
}

type CatalogHandler struct {
	catalogService CatalogService
}

func NewCatalogHandler(s CatalogService) *CatalogHandler {
	return &CatalogHandler{
		catalogService: s,
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
	id := c.Param("id")

	item, err := h.catalogService.GetCatalogItem(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *CatalogHandler) BuyItem(c *gin.Context) {
	itemID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"message": "purchase request accepted",
		"item_id": itemID,
	})
}
