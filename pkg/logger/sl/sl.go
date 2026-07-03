package sl

import (
	"context"
	"log/slog"
)

func Err(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "nil")
	}

	return slog.String("error", err.Error())
}

func LoggerWithCtx(ctx context.Context, logger *slog.Logger) *slog.Logger {
	reqID, ok := ctx.Value("request_id").(string)
	if !ok {
		return logger
	}

	return logger.With(slog.String("request_id", reqID))
}
