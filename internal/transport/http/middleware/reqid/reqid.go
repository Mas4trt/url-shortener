package reqid

import (
	"context"
	"net/http"

	sl "url-shortener/pkg/logger/sl"

	"github.com/go-chi/chi/v5/middleware"
)

// Propagate must be mounted after middleware.RequestID.
func Propagate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id := middleware.GetReqID(r.Context()); id != "" {
			ctx := context.WithValue(r.Context(), sl.RequestIDKey, id)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
