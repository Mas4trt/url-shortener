package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"url-shortener/pkg/logger/sl"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitPostgres(log *slog.Logger, dsn string) (*PostgresRepo, func(), error) {
	const op = "postgres.InitPosttgres"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Error("failed to configure postgres pool", sl.Err(err))
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		log.Error("failed to connect to postgres", sl.Err(err))
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	repo := &PostgresRepo{pool: pool}

	cleanup := func() {
		log.Info("closing postgres connection pool")
		repo.pool.Close()
	}

	log.Info("postgres connection established")
	return repo, cleanup, nil
}
