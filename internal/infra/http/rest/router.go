package rest

import (
	"avito-queue/internal/infra/http/rest/handlers"

	"github.com/go-chi/chi/v5"
)

func NewRouter() *chi.Mux {
	router := chi.NewRouter()

	router.Get("/health", handlers.HealthHandler)

	return router
}
