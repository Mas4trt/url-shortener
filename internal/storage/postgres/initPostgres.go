package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"url-shortener/pkg/logger/sl"
)

func InitPostgres(log *slog.Logger, dsn string) (*PostgresRepo, error) {
	const op = "postgres.InitPosttgres"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := New(ctx, dsn)
	if err != nil {
		log.Error("failed to connect postgres", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := db.pool.Ping(ctx); err != nil {
		db.pool.Close()
		log.Error("failed to connect postgres", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return db, nil
}
