package redis

import (
	"context"
	"errors"
	"log/slog"
	"time"
	"url-shortener/pkg/logger/sl"

	"golang.org/x/sync/singleflight"

	"github.com/redis/go-redis/v9"
)

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
}

func New(log *slog.Logger, next Storage, rdb *redis.Client, ttl time.Duration) *Cache {
	return &Cache{
		log:  log,
		next: next,
		rdb:  rdb,
		ttl:  ttl,
	}
}

func (c *Cache) Save(ctx context.Context, url string, alias string) error {
	if err := c.next.Save(ctx, url, alias); err != nil {
		return err
	}

	err := c.rdb.Set(ctx, alias, url, c.ttl).Err()
	if err != nil {
		c.log.Warn("failed to set cache on save", sl.Err(err))
	}

	return nil
}

func (c *Cache) Get(ctx context.Context, alias string) (string, error) {
	val, err := c.rdb.Get(ctx, alias).Result()
	if err == nil {
		return val, nil
	}

	if err != redis.Nil {
		c.log.Warn("redis down", sl.Err(err))
	}

	v, err, _ := c.group.Do(alias, func() (interface{}, error) {
		url, err := c.next.Get(ctx, alias)
		if err != nil {
			return "", err
		}

		err = c.rdb.Set(ctx, alias, url, c.ttl).Err()
		if err != nil {
			c.log.Warn("redis set failed", sl.Err(err))
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

	err := c.rdb.Del(ctx, alias).Err()
	if err != nil {
		c.log.Warn("failed to delete from cache", sl.Err(err))
	}
	return nil
}
