package app

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"avito-queue/internal/config"
)

const migrationsDir = "migrations"

type Repositories struct {
	pool *pgxpool.Pool
}

func newRepositories(ctx context.Context, cfg *config.Database) (*Repositories, error) {
	connstr := buildDSN(cfg)

	if err := runMigrations(ctx, connstr); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	pool, err := pgxpool.New(ctx, connstr)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Repositories{
		pool: pool,
	}, nil
}

func runMigrations(ctx context.Context, connstr string) error {
	db, err := sql.Open("pgx", connstr)
	if err != nil {
		return fmt.Errorf("open migrations connection: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, migrationsDir); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

func buildDSN(cfg *config.Database) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)
}
