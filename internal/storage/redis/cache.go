package redis

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"
	"url-shortener/pkg/logger/sl"

	"golang.org/x/sync/singleflight"

	"github.com/redis/go-redis/v9"
)

// redisCooldown is how long the breaker skips Redis after a failure.
// Deliberately short: this is a latency optimization for a degraded
// Redis, not a durability mechanism — Postgres remains the source of
// truth and every path here still falls back to it.
const redisCooldown = 5 * time.Second

type Storage interface {
	Save(ctx context.Context, url string, alias string) error
	Get(ctx context.Context, alias string) (string, error)
	Delete(ctx context.Context, alias string) error
}

type Cache struct {
	log   *slog.Logger
	next  Storage
	rdb   *redis.Client
	ttl   time.Duration
	group singleflight.Group

	// downUntil holds a UnixNano timestamp; while time.Now() is before
	// it, the breaker is open and Redis calls are skipped entirely.
	// Zero value means "never tripped" — always available.
	downUntil atomic.Int64
}

func New(log *slog.Logger, next Storage, rdb *redis.Client, ttl time.Duration) *Cache {
	return &Cache{
		log:  log,
		next: next,
		rdb:  rdb,
		ttl:  ttl,
	}
}

func (c *Cache) redisAvailable() bool {
	return time.Now().UnixNano() >= c.downUntil.Load()
}

func (c *Cache) markRedisDown() {
	c.downUntil.Store(time.Now().Add(redisCooldown).UnixNano())
}

func (c *Cache) Save(ctx context.Context, url string, alias string) error {
	if err := c.next.Save(ctx, url, alias); err != nil {
		return err
	}

	if !c.redisAvailable() {
		c.log.Debug("skipping cache write, breaker open", slog.String("alias", alias))
		return nil
	}

	if err := c.rdb.Set(ctx, alias, url, c.ttl).Err(); err != nil {
		c.log.Warn("failed to set cache on save", sl.Err(err))
		c.markRedisDown()
	}

	return nil
}

func (c *Cache) Get(ctx context.Context, alias string) (string, error) {
	if c.redisAvailable() {
		val, err := c.rdb.Get(ctx, alias).Result()
		if err == nil {
			return val, nil
		}

		if !errors.Is(err, redis.Nil) {
			c.log.Warn("redis unavailable, falling back to db", sl.Err(err))
			c.markRedisDown()
		}
	}

	v, err, _ := c.group.Do(alias, func() (interface{}, error) {
		url, err := c.next.Get(ctx, alias)
		if err != nil {
			return "", err
		}

		if c.redisAvailable() {
			if err := c.rdb.Set(ctx, alias, url, c.ttl).Err(); err != nil {
				c.log.Warn("redis set failed", sl.Err(err))
				c.markRedisDown()
			}
		}

		return url, nil
	})

	if err != nil {
		return "", err
	}

	res, ok := v.(string)
	if !ok {
		return "", errors.New("invalid type from singleflight")
	}
	return res, nil
}

func (c *Cache) Delete(ctx context.Context, alias string) error {
	if err := c.next.Delete(ctx, alias); err != nil {
		return err
	}

	if !c.redisAvailable() {
		c.log.Debug("skipping cache delete, breaker open", slog.String("alias", alias))
		return nil
	}

	if err := c.rdb.Del(ctx, alias).Err(); err != nil {
		c.log.Warn("failed to delete from cache", sl.Err(err))
		c.markRedisDown()
	}

	return nil
}
