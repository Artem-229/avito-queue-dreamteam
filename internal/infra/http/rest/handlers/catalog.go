package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"avito-queue/internal/domain"
)

type CatalogService interface {
	GetCatalog(ctx context.Context) ([]domain.CatalogItem, error)
	GetCatalogItem(ctx context.Context, id uuid.UUID) (domain.CatalogItem, error)
	GetSimilarItems(ctx context.Context, id uuid.UUID) ([]domain.CatalogItem, error)
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
		respondErrorCode(c, http.StatusInternalServerError, CodeCatalogFailure, "Не удалось загрузить каталог")
		return
	}

	if items == nil {
		items = []domain.CatalogItem{}
	}

	c.JSON(http.StatusOK, items)
}

func (h *CatalogHandler) GetByID(c *gin.Context) {
	itemID, ok := itemIDFromParam(c)
	if !ok {
		return
	}

	item, err := h.catalogService.GetCatalogItem(c.Request.Context(), itemID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *CatalogHandler) GetSimilarItems(c *gin.Context) {
	itemID, ok := itemIDFromParam(c)
	if !ok {
		return
	}

	items, err := h.catalogService.GetSimilarItems(c.Request.Context(), itemID)
	if err != nil {
		respondError(c, err)
		return
	}

	if items == nil {
		items = []domain.CatalogItem{}
	}

	c.JSON(http.StatusOK, items)
}
