package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/domain"
	"url-shortener/internal/transport/http/handlers"
	"url-shortener/internal/transport/http/handlers/mocks"
	"url-shortener/internal/transport/http/validation"
)

// setupTest инициализирует все необходимые компоненты для тестов
func setupTest(t *testing.T) (*mocks.URLService, *handlers.Handler) {
	t.Helper()
	mockService := mocks.NewURLService(t)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	v, _ := validation.New()

	handler := handlers.New(logger, mockService, v)

	return mockService, handler
}

func TestHandler_Save(t *testing.T) {
	tests := []struct {
		name         string
		payload      string
		mockBehavior func(s *mocks.URLService)
		expectedCode int
	}{
		{
			name:    "Success without custom alias",
			payload: `{"url": "https://google.com"}`,
			mockBehavior: func(s *mocks.URLService) {
				s.On("Save", mock.Anything, "https://google.com", "").
					Return("rndstr", nil).Once()
			},
			expectedCode: http.StatusOK,
		},
		{
			name:    "Success with custom alias",
			payload: `{"url": "https://google.com", "alias": "goog"}`,
			mockBehavior: func(s *mocks.URLService) {
				s.On("Save", mock.Anything, "https://google.com", "goog").
					Return("goog", nil).Once()
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "Invalid JSON",
			payload:      `{"url": "https://google.com"`,
			mockBehavior: func(s *mocks.URLService) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Validation error (invalid URL)",
			payload:      `{"url": "invalid-url"}`,
			mockBehavior: func(s *mocks.URLService) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:    "URL already exists",
			payload: `{"url": "https://google.com", "alias": "goog"}`,
			mockBehavior: func(s *mocks.URLService) {
				s.On("Save", mock.Anything, "https://google.com", "goog").
					Return("", domain.ErrURLExist).Once()
			},
			expectedCode: http.StatusConflict,
		},
		{
			name:    "Internal service error",
			payload: `{"url": "https://google.com"}`,
			mockBehavior: func(s *mocks.URLService) {
				s.On("Save", mock.Anything, "https://google.com", "").
					Return("", errors.New("db connection lost")).Once()
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService, handler := setupTest(t)
			tt.mockBehavior(mockService)

			req := httptest.NewRequest(http.MethodPost, "/url", bytes.NewBufferString(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.Save(rr, req)

			require.Equal(t, tt.expectedCode, rr.Code)

			if tt.expectedCode == http.StatusOK {
				var resp handlers.SaveResponse
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				require.NotEmpty(t, resp.Alias)
			}
		})
	}
}

func TestHandler_Get(t *testing.T) {
	tests := []struct {
		name         string
		alias        string
		mockBehavior func(s *mocks.URLService)
		expectedCode int
		expectedURL  string
	}{
		{
			name:  "Success",
			alias: "goog",
			mockBehavior: func(s *mocks.URLService) {
				s.On("Get", mock.Anything, "goog").
					Return("https://google.com", nil).Once()
			},
			expectedCode: http.StatusFound, // 302
			expectedURL:  "https://google.com",
		},
		{
			name:  "URL not found",
			alias: "unknown",
			mockBehavior: func(s *mocks.URLService) {
				s.On("Get", mock.Anything, "unknown").
					Return("", domain.ErrURLNotFound).Once()
			},
			expectedCode: http.StatusNotFound, // 404
		},
		{
			name:  "Internal service error",
			alias: "error",
			mockBehavior: func(s *mocks.URLService) {
				s.On("Get", mock.Anything, "error").
					Return("", errors.New("some error")).Once()
			},
			expectedCode: http.StatusInternalServerError, // 500
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService, handler := setupTest(t)
			tt.mockBehavior(mockService)

			// Используем роутер chi для корректной передачи URLParam
			r := chi.NewRouter()
			r.Get("/{alias}", handler.Get)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.alias, nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tt.expectedCode, rr.Code)

			// Для 302 проверяем наличие правильного заголовка Location
			if tt.expectedCode == http.StatusFound {
				require.Equal(t, tt.expectedURL, rr.Header().Get("Location"))
			}
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	tests := []struct {
		name         string
		alias        string
		mockBehavior func(s *mocks.URLService)
		expectedCode int
	}{
		{
			name:  "Success delete",
			alias: "goog",
			mockBehavior: func(s *mocks.URLService) {
				s.On("Delete", mock.Anything, "goog").
					Return(nil).Once()
			},
			expectedCode: http.StatusNoContent, // 204
		},
		{
			name:  "URL not found on delete",
			alias: "unknown",
			mockBehavior: func(s *mocks.URLService) {
				s.On("Delete", mock.Anything, "unknown").
					Return(domain.ErrURLNotFound).Once()
			},
			expectedCode: http.StatusNotFound, // 404
		},
		{
			name:  "Internal service error on delete",
			alias: "error",
			mockBehavior: func(s *mocks.URLService) {
				s.On("Delete", mock.Anything, "error").
					Return(errors.New("db error")).Once()
			},
			expectedCode: http.StatusInternalServerError, // 500
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService, handler := setupTest(t)
			tt.mockBehavior(mockService)

			r := chi.NewRouter()
			r.Delete("/{alias}", handler.Delete)

			req := httptest.NewRequest(http.MethodDelete, "/"+tt.alias, nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tt.expectedCode, rr.Code)
		})
	}
}

// Отдельный тест для проверки пустого алиаса
func TestHandler_EmptyAliasEdgeCases(t *testing.T) {
	_, handler := setupTest(t)

	chiCtx := chi.NewRouteContext()
	reqCtx := context.WithValue(context.Background(), chi.RouteCtxKey, chiCtx)

	t.Run("Get with empty alias", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(reqCtx)
		rr := httptest.NewRecorder()

		handler.Get(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Delete with empty alias", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/", nil).WithContext(reqCtx)
		rr := httptest.NewRecorder()

		handler.Delete(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
