package redis_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"url-shortener/internal/storage/mocks"
	cache "url-shortener/internal/storage/redis"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newCache(t *testing.T, repo *mocks.URLRepository) (*cache.Cache, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	assert.NoError(t, err)

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	c := cache.New(discardLogger, repo, rdb, time.Second)

	t.Cleanup(func() {
		mr.Close()
		_ = rdb.Close()
	})

	return c, mr
}

func TestCache_Get_Hit(t *testing.T) {
	repo := mocks.NewURLRepository(t)
	c, mr := newCache(t, repo)

	ctx := context.Background()
	alias := "a1"
	val := "https://example.com"

	// preload redis
	mr.Set(alias, val)

	repo.AssertNotCalled(t, "Get", mock.Anything, alias)

	res, err := c.Get(ctx, alias)

	assert.NoError(t, err)
	assert.Equal(t, val, res)
}

func TestCache_Get_Miss_DB_OK(t *testing.T) {
	repo := mocks.NewURLRepository(t)
	c, mr := newCache(t, repo)

	ctx := context.Background()
	alias := "a1"
	val := "https://example.com"

	repo.
		On("Get", mock.Anything, alias).
		Return(val, nil).
		Once()

	res, err := c.Get(ctx, alias)

	assert.NoError(t, err)
	assert.Equal(t, val, res)

	// Redis must be filled
	v, _ := mr.Get(alias)
	assert.Equal(t, val, v)

	repo.AssertExpectations(t)
}

func TestCache_Get_Miss_DB_Error(t *testing.T) {
	repo := mocks.NewURLRepository(t)
	c, _ := newCache(t, repo)

	ctx := context.Background()
	alias := "a1"

	repo.
		On("Get", mock.Anything, alias).
		Return("", assert.AnError).
		Once()

	res, err := c.Get(ctx, alias)

	assert.Error(t, err)
	assert.Empty(t, res)

	repo.AssertExpectations(t)
}

func TestCache_Get_RedisMiss_DBFallback(t *testing.T) {
	repo := mocks.NewURLRepository(t)
	c, _ := newCache(t, repo)

	ctx := context.Background()
	alias := "a1"
	val := "https://example.com"

	repo.
		On("Get", mock.Anything, alias).
		Return(val, nil).
		Once()

	res, err := c.Get(ctx, alias)

	assert.NoError(t, err)
	assert.Equal(t, val, res)

	repo.AssertExpectations(t)
}

func TestCache_Get_Singleflight(t *testing.T) {
	repo := mocks.NewURLRepository(t)
	c, _ := newCache(t, repo)

	ctx := context.Background()
	alias := "a1"
	val := "https://example.com"

	repo.
		On("Get", mock.Anything, alias).
		Return(val, nil).
		Once()

	var wg sync.WaitGroup
	n := 10

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			res, err := c.Get(ctx, alias)
			assert.NoError(t, err)
			assert.Equal(t, val, res)
		}()
	}

	wg.Wait()

	repo.AssertNumberOfCalls(t, "Get", 1)
}
