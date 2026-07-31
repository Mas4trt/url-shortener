package logger_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	logger "url-shortener/internal/transport/http/middleware"

	"github.com/go-chi/chi/v5/middleware"
)

func TestLoggerMiddleware(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		path          string
		reqID         string
		handlerStatus int
		expectedLevel string
		expectedMsg   string
		expectLog     bool
	}{
		{
			name:          "Успешный запрос (200) - уровень INFO",
			method:        http.MethodGet,
			path:          "/api/v1/resource",
			reqID:         "req-123",
			handlerStatus: http.StatusOK,
			expectedLevel: "INFO",
			expectedMsg:   "request completed",
			expectLog:     true,
		},
		{
			name:          "Ошибка клиента (404) - уровень WARN",
			method:        http.MethodPost,
			path:          "/api/v1/not-found",
			handlerStatus: http.StatusNotFound,
			expectedLevel: "WARN",
			expectedMsg:   "request error",
			expectLog:     true,
		},
		{
			name:          "Ошибка сервера (500) - уровень ERROR",
			method:        http.MethodDelete,
			path:          "/api/v1/crash",
			handlerStatus: http.StatusInternalServerError,
			expectedLevel: "ERROR",
			expectedMsg:   "request failed",
			expectLog:     true,
		},
		{
			name:          "Пропуск логов для healthcheck",
			method:        http.MethodGet,
			path:          "/health",
			handlerStatus: http.StatusOK,
			expectLog:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var buf bytes.Buffer
			log := slog.New(slog.NewJSONHandler(&buf, nil))
			mw := logger.New(log)

			buf.Reset()

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.handlerStatus)
				_, _ = w.Write([]byte("response body"))
			})

			handler := mw(nextHandler)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.RemoteAddr = "127.0.0.1:54321"
			req.Header.Set("User-Agent", "Test-Agent/1.0")

			if tt.reqID != "" {
				ctx := req.Context()
				ctx = context.WithValue(ctx, middleware.RequestIDKey, tt.reqID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if !tt.expectLog {
				if buf.Len() > 0 {
					t.Errorf("ожидалось отсутствие логов для пути %s, но получено: %s", tt.path, buf.String())
				}
				return
			}

			if buf.Len() == 0 {
				t.Fatalf("ожидалась запись в лог, но буфер пуст")
			}

			var logEntry map[string]any
			scanner := bufio.NewScanner(&buf)
			for scanner.Scan() {
				if err := json.Unmarshal(scanner.Bytes(), &logEntry); err != nil {
					t.Fatalf("не удалось распарсить JSON лога: %v", err)
				}
			}

			if level, ok := logEntry["level"]; !ok || level != tt.expectedLevel {
				t.Errorf("ожидался level %q, получено: %v", tt.expectedLevel, level)
			}
			if msg, ok := logEntry["msg"]; !ok || msg != tt.expectedMsg {
				t.Errorf("ожидался msg %q, получено: %v", tt.expectedMsg, msg)
			}

			if status, ok := logEntry["status"]; !ok || status != float64(tt.handlerStatus) {
				t.Errorf("ожидался status %d, получено: %v", tt.handlerStatus, status)
			}
			if bytesWritten, ok := logEntry["bytes"]; !ok || bytesWritten.(float64) <= 0 {
				t.Errorf("ожидалось количество записанных байт > 0, получено: %v", bytesWritten)
			}
			if _, ok := logEntry["duration"]; !ok {
				t.Errorf("ожидалось наличие поля 'duration'")
			}

			reqGroup, ok := logEntry["request"].(map[string]any)
			if !ok {
				t.Fatalf("ожидалось наличие группы 'request' в формате JSON объекта")
			}

			if method, ok := reqGroup["method"]; !ok || method != tt.method {
				t.Errorf("в группе request ожидался method %q, получено: %v", tt.method, method)
			}
			if path, ok := reqGroup["path"]; !ok || path != tt.path {
				t.Errorf("в группе request ожидался path %q, получено: %v", tt.path, path)
			}
			if agent, ok := reqGroup["user_agent"]; !ok || agent != "Test-Agent/1.0" {
				t.Errorf("ожидался user_agent 'Test-Agent/1.0', получено: %v", agent)
			}

			if tt.reqID != "" {
				if id, ok := reqGroup["request_id"]; !ok || id != tt.reqID {
					t.Errorf("ожидался request_id %q, получено: %v", tt.reqID, id)
				}
			}

		})
	}
}
