package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"url-shortener/pkg/logger/sl"

	"github.com/redis/go-redis/v9"
)

func InitRedis(log *slog.Logger, addr string) (*redis.Client, func(), error) {
	const op = "redis.InitRedis"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		PoolSize:     10,
		MinIdleConns: 2,
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
