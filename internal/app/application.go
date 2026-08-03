package app

import (
	"avito-queue/internal/infra/http/rest"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"avito-queue/internal/config"
)

const shutdownTimeout = 5 * time.Second

type App struct {
	pool       *pgxpool.Pool
	httpServer *rest.Server
}

func New(ctx context.Context, conf *config.Config) (*App, error) {
	pool, err := newPostgresPool(ctx, conf.Database)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	if err := runMigrations(ctx, conf.Database); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	server := rest.NewServer(conf)

	return &App{
		pool:       pool,
		httpServer: server,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
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
		a.pool.Close()
		return err
	case <-ctx.Done():
		_, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		a.pool.Close()
		return nil
	}
}
