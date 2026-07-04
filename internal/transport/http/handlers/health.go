package handlers

import (
	"context"
	"net/http"

	"github.com/redis/go-redis/v9"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

func ReadinessProbe(db Pinger, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if err := db.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("db down"))
			return
		}

		if err := rdb.Ping(ctx).Err(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("redis down"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}
