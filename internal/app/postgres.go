package app

import (
	purchaseright "avito-queue/internal/infra/postgres/repository/purchase_right"
	"context"
	"fmt"

	"avito-queue/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Repositories struct {
	PurchaseRightRepo *purchaseright.RightRepo
	pool              *pgxpool.Pool
}

func NewRepositories(ctx context.Context, cfg *config.Database) (*Repositories, error) {
	connstr := buildDSN(cfg)

	pool, err := pgxpool.New(ctx, connstr)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	purchaseRightRepo := purchaseright.NewRightRepo(pool)

	return &Repositories{
		PurchaseRightRepo: purchaseRightRepo,
		pool:              pool,
	}, nil
}

func buildDSN(cfg *config.Database) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)
}
