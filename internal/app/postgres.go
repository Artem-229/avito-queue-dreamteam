package app

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	"avito-queue/internal/config"
	"avito-queue/internal/infra/postgres/repository"
)

type Repositories struct {
	QueueEntryRepo    *repository.QueueEntryRepository
	CatalogRepository *repository.CatalogRepository
	StatsRepo         *repository.StatsRepository
	pool              *pgxpool.Pool
}

// Pool отдаёт пул для проверок живости (/health).
func (r *Repositories) Pool() *pgxpool.Pool { return r.pool }

const (
	maxConns          = 30
	minConns          = 5
	maxConnLifetime   = time.Hour
	maxConnIdleTime   = 5 * time.Minute
	connectionTimeout = 5 * time.Second
)

func NewRepositories(ctx context.Context, cfg *config.Database) (*Repositories, error) {
	poolCfg, err := pgxpool.ParseConfig(buildDSN(cfg))
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}

	poolCfg.MaxConns = maxConns
	poolCfg.MinConns = minConns
	poolCfg.MaxConnLifetime = maxConnLifetime
	poolCfg.MaxConnIdleTime = maxConnIdleTime
	poolCfg.ConnConfig.ConnectTimeout = connectionTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	queueEntryRepo := repository.NewQueueEntryRepo(pool)
	catalogRepository := repository.NewCatalogRepository(pool)
	statsRepo := repository.NewStatsRepository(pool)

	return &Repositories{
		QueueEntryRepo:    queueEntryRepo,
		CatalogRepository: catalogRepository,
		StatsRepo:         statsRepo,
		pool:              pool,
	}, nil
}

func buildDSN(cfg *config.Database) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)
}
