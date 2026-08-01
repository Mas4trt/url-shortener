//go:build integration
// +build integration

package integration

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	dbpostgres "url-shortener/internal/storage/postgres"
	cacheredis "url-shortener/internal/storage/redis"
)

type dbSpyStorage struct {
	next  cacheredis.Storage
	calls atomic.Int32
}

func (s *dbSpyStorage) Save(ctx context.Context, url string, alias string) error {
	return s.next.Save(ctx, url, alias)
}

func (s *dbSpyStorage) Get(ctx context.Context, alias string) (string, error) {
	s.calls.Add(1)
	time.Sleep(50 * time.Millisecond)
	return s.next.Get(ctx, alias)
}

func (s *dbSpyStorage) Delete(ctx context.Context, alias string) error {
	return s.next.Delete(ctx, alias)
}

func (s *dbSpyStorage) Calls() int32 {
	return s.calls.Load()
}

func (s *IntegrationSuite) TestCacheLayer_HitAndMiss() {
	s.SetupTest() // Очищаем БД и Redis

	pgRepo, err := dbpostgres.New(s.ctx, s.cfg.DatabaseURL)
	s.Require().NoError(err)
	defer pgRepo.Close()

	cacheLayer := cacheredis.New(slog.Default(), pgRepo, s.redis, time.Minute)

	s.Run("Cache Miss - fetches from DB and stores in Cache", func() {
		err := pgRepo.Save(s.ctx, "https://database-only.com", "miss-alias")
		s.Require().NoError(err)

		url, err := cacheLayer.Get(s.ctx, "miss-alias")
		s.Require().NoError(err)
		s.Require().Equal("https://database-only.com", url)

		redisVal, err := s.redis.Get(s.ctx, "miss-alias").Result()
		s.Require().NoError(err)
		s.Require().Equal("https://database-only.com", redisVal)
	})

	s.Run("Cache Hit - returns from Cache, ignores DB", func() {
		err := s.redis.Set(s.ctx, "hit-alias", "https://redis-fast.com", time.Minute).Err()
		s.Require().NoError(err)

		url, err := cacheLayer.Get(s.ctx, "hit-alias")
		s.Require().NoError(err)

		s.Require().Equal("https://redis-fast.com", url)
	})
}

func (s *IntegrationSuite) TestCacheLayer_Singleflight() {
	s.SetupTest()

	pgRepo, err := dbpostgres.New(s.ctx, s.cfg.DatabaseURL)
	s.Require().NoError(err)
	defer pgRepo.Close()

	err = pgRepo.Save(s.ctx, "https://highload.com", "viral-link")
	s.Require().NoError(err)

	spyDB := &dbSpyStorage{next: pgRepo}
	cacheLayer := cacheredis.New(slog.Default(), spyDB, s.redis, time.Minute)

	const workers = 100
	var wg sync.WaitGroup
	var errs atomic.Int32

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			url, err := cacheLayer.Get(s.ctx, "viral-link")
			if err != nil || url != "https://highload.com" {
				errs.Add(1)
			}
		}()
	}

	wg.Wait()

	s.Require().Equal(int32(0), errs.Load(), "All concurrent requests should succeed")
	s.Require().Equal(int32(1), spyDB.Calls(), "Singleflight failed: expected exactly 1 DB call")
}
