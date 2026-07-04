package sl_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"url-shortener/pkg/logger/sl"
)

func TestErr(t *testing.T) {
	sampleErr := errors.New("some standard error")

	tests := []struct {
		name     string
		err      error
		expected slog.Attr
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: slog.Attr{},
		},
		{
			name:     "standard error",
			err:      sampleErr,
			expected: slog.Any("error", sampleErr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := sl.Err(tt.err)

			if actual.Key != tt.expected.Key {
				t.Errorf("expected key %q, got %q", tt.expected.Key, actual.Key)
			}

			if actual.Value.Any() != tt.expected.Value.Any() {
				t.Errorf("expected value %v, got %v", tt.expected.Value.Any(), actual.Value.Any())
			}
		})
	}
}

func TestLoggerWithCtx(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey || a.Key == slog.LevelKey {
				return slog.Attr{}
			}
			return a
		},
	})
	baseLogger := slog.New(handler)

	tests := []struct {
		name         string
		ctx          context.Context
		wantContains string
		wantMissing  string
	}{
		{
			name:         "context with valid request_id string",
			ctx:          context.WithValue(context.Background(), sl.RequestIDKey, "test-req-id-123"),
			wantContains: "request_id=test-req-id-123",
		},
		{
			name:        "empty context",
			ctx:         context.Background(),
			wantMissing: "request_id",
		},
		{
			name:        "context with wrong request_id type (e.g. int instead of string)",
			ctx:         context.WithValue(context.Background(), sl.RequestIDKey, 12345),
			wantMissing: "request_id",
		},
		{
			name: "context with unrelated value",
			ctx: func() context.Context {
				type dummyKey string
				return context.WithValue(context.Background(), dummyKey("other_key"), "some_value")
			}(),
			wantMissing: "request_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			logger := sl.LoggerWithCtx(tt.ctx, baseLogger)

			logger.Info("test message")

			output := buf.String()

			if tt.wantContains != "" && !strings.Contains(output, tt.wantContains) {
				t.Errorf("expected log output to contain %q, but got: %q", tt.wantContains, output)
			}

			if tt.wantMissing != "" && strings.Contains(output, tt.wantMissing) {
				t.Errorf("expected log output NOT to contain %q, but got: %q", tt.wantMissing, output)
			}
		})
	}
}
