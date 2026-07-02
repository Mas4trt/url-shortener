package logger

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func New(log *slog.Logger) func(next http.Handler) http.Handler {
	logger := log.With(slog.String("component", "middleware/Logger"))
	logger.Info("logger middleware enabled")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqArgs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			}

			if reqID := middleware.GetReqID(r.Context()); reqID != "" {
				reqArgs = append(reqArgs, slog.String("request_id", reqID))
			}

			entry := logger.With(reqArgs...)
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				status := ww.Status()

				resArgs := []any{
					slog.Int("status", status),
					slog.Int("bytes", ww.BytesWritten()),
					slog.Duration("duration", time.Since(start)),
				}

				switch {
				case status >= 500:
					entry.Error("request failed", resArgs...)
				case status >= 400:
					entry.Warn("request error", resArgs...)
				default:
					entry.Info("request completed", resArgs...)
				}
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
