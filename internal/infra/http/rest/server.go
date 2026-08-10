package rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"avito-queue/internal/config"
	"avito-queue/internal/infra/http/rest/handlers"
)

// Таймауты входящих соединений. Без них клиент, открывший соединение и не
// дославший запрос, держит горутину и файловый дескриптор бесконечно: это и
// slowloris, и куда более частый случай — телефон, потерявший сеть посреди
// запроса, или прокси, оборвавший TCP без FIN. Сервис от этого не падает, он
// тихо перестаёт принимать новые соединения.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
)

type Server struct {
	server *http.Server
}

func NewServer(
	conf *config.Config,
	catalogService handlers.CatalogService,
	queueService handlers.QueueService,
	purchaseRightService handlers.PurchaseRightService,
	demoService handlers.DemoService,
	statsService handlers.StatsService,
	db handlers.Pinger,
	logger *slog.Logger,
) *Server {
	h := handlers.New(
		handlers.NewCatalogHandler(catalogService),
		handlers.NewQueueHandler(queueService, purchaseRightService),
		handlers.NewDemoHandler(demoService),
		handlers.NewStatsHandler(statsService),
		db,
	)

	router := NewRouter(h, conf, logger)

	return &Server{
		server: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", conf.HTTPServer.Host, conf.HTTPServer.Port),
			Handler:           router,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
	}
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
