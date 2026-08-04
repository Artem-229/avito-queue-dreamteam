package rest

import (
	"avito-queue/internal/config"
	"avito-queue/internal/infra/http/rest/handlers"
	"context"
	"fmt"
	"net/http"
)

type Server struct {
	server *http.Server
}

func NewServer(conf *config.Config) *Server {
	handlers := handlers.New()

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
