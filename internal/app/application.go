package app

import (
	"avito-queue/internal/infra/http/rest"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"avito-queue/internal/config"

	"github.com/lmittmann/tint"
)

const shutdownTimeout = 5 * time.Second

type App struct {
	repositories *Repositories
	httpServer   *rest.Server
	logger       *slog.Logger
}

func New(ctx context.Context, conf *config.Config) (*App, error) {
	logger := slog.New(tint.NewTextHandler(os.Stdout, &tint.Options{
		Level:      slog.Level(conf.Logger.Level),
		TimeFormat: time.DateTime,
	}))

	repos, err := newRepositories(ctx, conf.Database)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	server := rest.NewServer(conf)

	return &App{
		repositories: repos,
		httpServer:   server,
		logger:       logger,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("starting application")

	errChan := make(chan error, 1)

	go func() {
		if err := a.httpServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("rest start: %w", err)
			return
		}
		errChan <- nil
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		a.logger.Info("shutting down, please wait...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := a.httpServer.Stop(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}

		a.repositories.pool.Close()

		return nil
	}
}
