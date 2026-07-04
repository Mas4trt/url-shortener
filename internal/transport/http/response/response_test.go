package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
		OtherField int    `validate:"min=10"`
	}

	tests := []struct {
		name      string
		input     TestStruct
		wantError string
	}{
		{
			name: "required field missing",
			input: TestStruct{
				URLField:   "https://example.com",
				OtherField: 15,
			},
			wantError: "field ReqField is a required field",
		},
		{
			name: "invalid url",
			input: TestStruct{
				ReqField:   "present",
				URLField:   "not-a-valid-url",
				OtherField: 15,
			},
			wantError: "field URLField is not valid URL",
		},
		{
			name: "default case (min validation failed)",
			input: TestStruct{
				ReqField:   "present",
				URLField:   "https://example.com",
				OtherField: 5,
			},
			wantError: "field OtherField is not valid",
		},
		{
			name: "multiple validation errors",
			input: TestStruct{
				URLField:   "bad-url",
				OtherField: 5,
			},
			wantError: "field ReqField is a required field, field URLField is not valid URL, field OtherField is not valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.input)

			require.Error(t, err)
			var valErrs validator.ValidationErrors
			require.ErrorAs(t, err, &valErrs, "Error should be of type validator.ValidationErrors")

			resp := ValidationError(valErrs)

			assert.Equal(t, StatusError, resp.Status)

			expectedErrors := strings.Split(tt.wantError, ", ")
			for _, expectedErr := range expectedErrors {
				assert.Contains(t, resp.Error, expectedErr)
			}
		})
	}
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
