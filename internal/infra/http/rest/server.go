package rest

import (
	"avito-queue/internal/config"
	"net/http"
)

type Server struct {
	Config  *config.Config
	Handler http.Handler
}

type ServerDeps struct {
	Config  *config.Config
	Handler http.Handler
}

func NewServer(conf *config.Config) *Server {
	router := NewRouter()
	return &Server{
		Config:  conf,
		Handler: router,
	}
}

func (s *Server) Start() error {
	return http.ListenAndServe(s.Config.HTTPServer.Host, s.Handler)
}
