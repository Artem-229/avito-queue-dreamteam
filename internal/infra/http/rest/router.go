package rest

import (
	"avito-queue/internal/infra/http/rest/handlers"

	"github.com/go-chi/chi/v5"
)

func NewRouter(h *handlers.Handlers) *chi.Mux {
	router := chi.NewRouter()

	router.Get("/health", h.Health)

	return router
}
