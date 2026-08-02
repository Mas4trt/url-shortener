package response_test

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"url-shortener/internal/transport/http/response"

	"github.com/stretchr/testify/require"
)

func TestRespondError(t *testing.T) {
	errSentinelA := errors.New("sentinel a")
	errSentinelB := errors.New("sentinel b")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cases := []response.ErrCase{
		response.Case(errSentinelA, http.StatusConflict, "case a"),
		response.Case(errSentinelB, http.StatusNotFound, "case b"),
	}

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "matches a case directly",
			err:        errSentinelA,
			wantStatus: http.StatusConflict,
			wantMsg:    "case a",
		},
		{
			name:       "matches a case through wrapping",
			err:        fmt.Errorf("op failed: %w", errSentinelB),
			wantStatus: http.StatusNotFound,
			wantMsg:    "case b",
		},
		{
			name:       "falls back to the default when nothing matches",
			err:        errors.New("something unrelated"),
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			response.RespondError(rr, req, logger, tt.err, http.StatusInternalServerError, "fallback", cases...)

			require.Equal(t, tt.wantStatus, rr.Code)
			require.Contains(t, rr.Body.String(), tt.wantMsg)
		})
	}
}
