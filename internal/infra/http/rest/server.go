package rest

import (
	"avito-queue/internal/config"
	"avito-queue/internal/infra/http/rest/handlers"
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

type Server struct {
	server *http.Server
}

func NewServer(
	conf *config.Config,
	catalogService handlers.CatalogService,
	queueService handlers.QueueService,
	purchaseRightService handlers.PurchaseRightService,
	checkoutService handlers.CheckAccessor,
	logger *slog.Logger,
) *Server {
	catalogHandler := handlers.NewCatalogHandler(catalogService, purchaseRightService)
	queueHandler := handlers.NewQueueHandler(queueService)
	checkoutHandler := handlers.NewCheckoutHandler(checkoutService)

	h := handlers.New(catalogHandler, queueHandler, checkoutHandler)

	router := NewRouter(h, logger)
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
