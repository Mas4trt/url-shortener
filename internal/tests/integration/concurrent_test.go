package integration

import (
	"net/http"
	"sync"
	"sync/atomic"
)

func (s *IntegrationSuite) TestConcurrent_CreateAliasRaceCondition() {
	const (
		workers     = 50
		targetAlias = "super-sale"
		targetURL   = "https://shop.example.com/promo"
	)

	reqBody := map[string]string{
		"url":   targetURL,
		"alias": targetAlias,
	}

	var wg sync.WaitGroup
	var (
		successCount  atomic.Int32
		conflictCount atomic.Int32
		otherErrors   atomic.Int32
	)

	startGun := make(chan struct{})

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			<-startGun

			req := s.NewRequest(http.MethodPost, "/url", reqBody)
			req.Header.Set("Content-Type", "application/json")

			resp := s.Do(req)
			defer resp.Body.Close()

			switch resp.StatusCode {
			case http.StatusCreated, http.StatusOK:
				successCount.Add(1)
			case http.StatusConflict:
				conflictCount.Add(1)
			default:
				otherErrors.Add(1)
			}
		}()
	}

	close(startGun)

	wg.Wait()

	s.Require().Equal(int32(0), otherErrors.Load(), "Should not be any unexpected errors (500s, etc)")

	s.Require().Equal(int32(1), successCount.Load(), "Exactly ONE request should succeed in creating the alias")

	s.Require().Equal(int32(workers-1), conflictCount.Load(), "All other requests MUST be rejected with a conflict status")

	var count int
	err := s.db.QueryRow(s.ctx, "SELECT COUNT(*) FROM urlshortener.url WHERE alias = $1", targetAlias).Scan(&count)
	s.Require().NoError(err)
	s.Require().Equal(1, count, "Database MUST contain exactly 1 record for this alias")
}
