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
	}

	if items == nil {
		items = []domain.CatalogItem{}
	}
	c.JSON(http.StatusOK, items)
}

func (h *CatalogHandler) BuyItem(c *gin.Context) {
	itemID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"message": "purchase request accepted",
		"item_id": itemID,
	})
}
