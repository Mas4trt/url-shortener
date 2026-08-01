package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"url-shortener/pkg/logger/sl"

	"github.com/redis/go-redis/v9"
)

// Client tuning, kept as constants for the same reason as the Postgres
// pool settings — see storage/postgres/initPostgres.go's TODO. Promote
// to config.Config together with those once the wire graph is safe to
// regenerate.
const (
	dialTimeout  = 3 * time.Second
	readTimeout  = 2 * time.Second
	writeTimeout = 2 * time.Second
	pingTimeout  = 3 * time.Second
)

func InitRedis(log *slog.Logger, addr string) (*redis.Client, func(), error) {
	const op = "redis.InitRedis"

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		PoolSize:     10,
		MinIdleConns: 2,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error("redis not available", sl.Err(err))
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	cleanup := func() {
		log.Info("closing redis connection")
		if err := rdb.Close(); err != nil {
			log.Error("failed to close redis connection", sl.Err(err))
		}
	}

	log.Info("redis initialized")
	return rdb, cleanup, nil
}
