package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"url-shortener/pkg/logger/sl"

	"github.com/redis/go-redis/v9"
)

func InitRedis(log *slog.Logger, addr string) (*redis.Client, error) {
	const op = "redis.InitRedis"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error("redis not available", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("redis initialized")

	return rdb, nil
}
