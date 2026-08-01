package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"url-shortener/pkg/logger/sl"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool tuning. These are deliberately explicit rather than left to pgx's
// defaults (which size the pool relative to GOMAXPROCS — fine for a
// throwaway script, not something you want implicitly deciding how many
// connections a production replica opens against a shared database).
//
// TODO: promote these to config.Config fields once you're ready to touch
// the wire graph — right now they're constants so this change doesn't
// also require regenerating internal/bootstrap/wire_gen.go by hand.
const (
	poolMaxConns          = 20
	poolMinConns          = 2
	poolMaxConnLifetime   = time.Hour
	poolMaxConnIdleTime   = 15 * time.Minute
	poolHealthCheckPeriod = time.Minute
	connectTimeout        = 5 * time.Second
)

func InitPostgres(log *slog.Logger, dsn string) (*PostgresRepo, func(), error) {
	const op = "postgres.InitPostgres"

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Error("failed to parse postgres dsn", sl.Err(err))
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	poolCfg.MaxConns = poolMaxConns
	poolCfg.MinConns = poolMinConns
	poolCfg.MaxConnLifetime = poolMaxConnLifetime
	poolCfg.MaxConnIdleTime = poolMaxConnIdleTime
	poolCfg.HealthCheckPeriod = poolHealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
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

	log.Info("postgres connection established",
		slog.Int("max_conns", int(poolCfg.MaxConns)),
		slog.Int("min_conns", int(poolCfg.MinConns)),
	)
	return repo, cleanup, nil
}
