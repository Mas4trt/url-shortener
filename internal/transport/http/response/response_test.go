package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOk(t *testing.T) {
	resp := Ok()

	assert.Equal(t, StatusOk, resp.Status)
	assert.Empty(t, resp.Error, "Error field should be empty on OK response")
}

func TestError(t *testing.T) {
	errMsg := "something went wrong"
	resp := Error(errMsg)

	assert.Equal(t, StatusError, resp.Status)
	assert.Equal(t, errMsg, resp.Error)
}

func TestValidationError(t *testing.T) {
	validate := validator.New()

	type TestStruct struct {
		ReqField   string `validate:"required"`
		URLField   string `validate:"url"`
		OtherField string `validate:"min=10"`
	}

	t.Run("required", func(t *testing.T) {
		err := validate.Struct(TestStruct{})

		var valErrs validator.ValidationErrors
		require.ErrorAs(t, err, &valErrs)

		resp := ValidationError(valErrs)

		require.Equal(t, StatusError, resp.Status)
		require.Equal(t, "validation failed", resp.Error)
		require.Len(t, resp.Details, 3)

		assert.Equal(t, "reqfield", resp.Details[0].Field)
		assert.Equal(t, "is required", resp.Details[0].Message)
	})
}

func TestRespond(t *testing.T) {
	tests := []struct {
		name           string
		code           int
		payload        any
		wantStatus     string
		wantHTTPStatus int
	}{
		{
			name:           "successful response",
			code:           http.StatusOK,
			payload:        Ok(),
			wantStatus:     StatusOk,
			wantHTTPStatus: http.StatusOK,
		},
		{
			name:           "error response",
			code:           http.StatusBadRequest,
			payload:        Error("bad request"),
			wantStatus:     StatusError,
			wantHTTPStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/dummy-url", nil)

			Respond(recorder, request, tt.code, tt.payload)

			assert.Equal(t, tt.wantHTTPStatus, recorder.Code)

			assert.Contains(t, recorder.Header().Get("Content-Type"), "application/json")

			var responseBody Response
			err := json.Unmarshal(recorder.Body.Bytes(), &responseBody)
			require.NoError(t, err, "Response body should be valid JSON")

			assert.Equal(t, tt.wantStatus, responseBody.Status)

			if expectedPayload, ok := tt.payload.(Response); ok {
				assert.Equal(t, expectedPayload.Error, responseBody.Error)
			}
		})
	}
}
