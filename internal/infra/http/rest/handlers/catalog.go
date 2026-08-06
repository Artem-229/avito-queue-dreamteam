package handlers

import (
	"avito-queue/internal/domain"
	"avito-queue/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CatalogHandler struct {
	catalogService service.CatalogService
}

func NewCatalogHandler(s service.CatalogService) *CatalogHandler {
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
