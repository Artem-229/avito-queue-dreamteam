package rest

import (
	"avito-queue/internal/config"
	"avito-queue/internal/infra/http/rest/handlers"
	"avito-queue/internal/repository"
	"avito-queue/internal/service"
	"avito-queue/internal/usecases"
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	server *http.Server
}

func NewServer(conf *config.Config, db *pgxpool.Pool) *Server {
	catalogRepo := repository.NewCatalogRepository(db)
	catalogService := service.NewCatalogService(catalogRepo)
	CatalogHandler := handlers.NewCatalogHandler(catalogService)

	purchaseRightRepo := repository.NewPurchaseRightRepo(db)
	checkoutUsecase := usecases.NewCheckoutUsecase(purchaseRightRepo)
	checkoutHandler := &handlers.CheckoutHandler{Usecase: checkoutUsecase}

	handlers := handlers.New(CatalogHandler, checkoutHandler)

	router := NewRouter(handlers)
	addr := fmt.Sprintf("%s:%d", conf.HTTPServer.Host, conf.HTTPServer.Port)
	return &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: router,
		},
	}
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
