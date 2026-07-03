package sl

import (
	"context"
	"log/slog"
	"url-shortener/internal/domain"
)

func Err(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "nil")
	}

	return slog.String("error", err.Error())
}

func LoggerWithCtx(ctx context.Context, logger *slog.Logger) *slog.Logger {
	reqID, ok := ctx.Value(domain.RequestIDKey).(string)
	if !ok {
		return logger
	}

	return logger.With(slog.String("request_id", reqID))
}
