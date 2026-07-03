package sl

import (
	"context"
	"log/slog"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

func Err(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}

	return slog.Any("error", err)
}

func LoggerWithCtx(ctx context.Context, logger *slog.Logger) *slog.Logger {
	reqID, ok := ctx.Value(RequestIDKey).(string)
	if !ok {
		return logger
	}

	return logger.With(slog.String("request_id", reqID))
}
