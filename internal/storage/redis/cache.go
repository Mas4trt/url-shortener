package cache

import (
	"context"
	"time"
	service "url-shortener/internal/service/url"
	"url-shortener/pkg/logger/sl"

	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

type Cache struct {
	next service.URLRepository
	rdb  *redis.Client
	ttl  time.Duration
}

func New(next service.URLRepository, rdb *redis.Client, ttl time.Duration) *Cache {
	return &Cache{
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
		log.Warn("failed to set cache on save", sl.Err(err))
	}

	return nil
}

func (c *Cache) Get(ctx context.Context, alias string) (string, error) {
	val, err := c.rdb.Get(ctx, alias).Result()
	if err == nil {
		return val, nil
	}

	if err != redis.Nil && err != nil {
		log.Warn("redis get failed", sl.Err(err))
	}

	url, err := c.next.Get(ctx, alias)
	if err != nil {
		return "", err
	}

	err = c.rdb.Set(ctx, alias, url, c.ttl).Err()
	if err != nil {
		log.Warn("redis set failed", sl.Err(err))
	}
	return url, nil
}

func (c *Cache) Delete(ctx context.Context, alias string) error {
	if err := c.next.Delete(ctx, alias); err != nil {
		return err
	}

	err := c.rdb.Del(ctx, alias).Err()
	if err != nil {
		log.Warn("failed to delete from cache", sl.Err(err))
	}
	return nil
}
