package redis_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	cache "url-shortener/internal/storage/redis"
	"url-shortener/internal/storage/redis/mocks"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newCache(t *testing.T, repo *mocks.Storage) (*cache.Cache, *miniredis.Miniredis) {
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
	repo := mocks.NewStorage(t)
	c, mr := newCache(t, repo)

	ctx := context.Background()
	alias := "a1"
	val := "https://example.com"

	require.NoError(t, mr.Set(alias, val))

	repo.AssertNotCalled(t, "Get", mock.Anything, alias)

	res, err := c.Get(ctx, alias)

	assert.NoError(t, err)
	assert.Equal(t, val, res)
}

func TestCache_Get_Miss_DB_OK(t *testing.T) {
	repo := mocks.NewStorage(t)
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

	v, _ := mr.Get(alias)
	assert.Equal(t, val, v)

	repo.AssertExpectations(t)
}

func TestCache_Get_Miss_DB_Error(t *testing.T) {
	repo := mocks.NewStorage(t)
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
	repo := mocks.NewStorage(t)
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
	repo := mocks.NewStorage(t)
	c, _ := newCache(t, repo)

	ctx := context.Background()
	alias := "a1"
	val := "https://example.com"

	started := make(chan struct{})
	release := make(chan struct{})

	repo.
		On("Get", mock.Anything, alias).
		Run(func(args mock.Arguments) {
			close(started)
			<-release
		}).
		Return(val, nil).
		Once()

	const workers = 10

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()

			res, err := c.Get(ctx, alias)

			assert.NoError(t, err)
			assert.Equal(t, val, res)
		}()
	}

	<-started

	time.Sleep(10 * time.Millisecond)

	close(release)

	wg.Wait()

	repo.AssertNumberOfCalls(t, "Get", 1)
}

// TestCache_Breaker_SkipsRedisAfterFailure verifies that once Redis
// becomes unreachable, the cache still serves requests correctly via the
// DB fallback — both immediately (the call that discovers the failure)
// and on subsequent calls (the breaker-open path that skips Redis
// entirely rather than retrying a dead connection on every request).
func TestCache_Breaker_SkipsRedisAfterFailure(t *testing.T) {
	repo := mocks.NewStorage(t)
	c, mr := newCache(t, repo)

	ctx := context.Background()
	alias := "a1"
	val := "https://example.com"

	mr.Close() // simulate Redis becoming unreachable

	repo.On("Get", mock.Anything, alias).Return(val, nil)

	res, err := c.Get(ctx, alias)
	require.NoError(t, err)
	require.Equal(t, val, res)

	// Breaker should now be open: this call must not hang or error trying
	// to reach the now-closed Redis instance.
	res, err = c.Get(ctx, alias)
	require.NoError(t, err)
	require.Equal(t, val, res)

	repo.On("Save", mock.Anything, "https://other.example.com", "a2").Return(nil).Once()
	require.NoError(t, c.Save(ctx, "https://other.example.com", "a2"),
		"Save must succeed even though the cache write is skipped")

	repo.On("Delete", mock.Anything, "a2").Return(nil).Once()
	require.NoError(t, c.Delete(ctx, "a2"),
		"Delete must succeed even though the cache delete is skipped")
}
