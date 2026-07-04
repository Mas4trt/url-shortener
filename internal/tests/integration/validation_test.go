package integration

import (
	"net/http"
)

func (s *IntegrationSuite) TestCreateURL_Validation() {

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
		expectedField  string
	}{
		{
			name: "Error - Missing URL",
			payload: map[string]interface{}{
				"alias": "my-link",
			},
			expectedStatus: http.StatusBadRequest,
			expectedField:  "url",
		},
		{
			name: "Error - Invalid URL Format",
			payload: map[string]interface{}{
				"url":   "not-a-valid-url",
				"alias": "my-link",
			},
			expectedStatus: http.StatusBadRequest,
			expectedField:  "url",
		},
		{
			name: "Error - Alias too short",
			payload: map[string]interface{}{
				"url":   "https://go.dev",
				"alias": "a",
			},
			expectedStatus: http.StatusBadRequest,
			expectedField:  "alias",
		},
		{
			name: "Error - Alias contains invalid characters",
			payload: map[string]interface{}{
				"url":   "https://go.dev",
				"alias": "my alias!",
			},
			expectedStatus: http.StatusBadRequest,
			expectedField:  "alias",
		},
		{
			name:           "Error - Empty Payload",
			payload:        map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			expectedField:  "url",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {

			req := s.NewRequest(http.MethodPost, "/url", tt.payload)
			req.Header.Set("Content-Type", "application/json")

			resp := s.Do(req)
			defer resp.Body.Close()

			s.Require().Equal(tt.expectedStatus, resp.StatusCode, "Expected HTTP status to match")

			var errResp struct {
				Error   string `json:"error"`
				Details []struct {
					Field   string `json:"field"`
					Message string `json:"message"`
				} `json:"details"`
			}

			s.DecodeBody(resp, &errResp)

			s.Require().NotEmpty(errResp.Error, "Error message should not be empty")

			found := false
			for _, detail := range errResp.Details {
				if detail.Field == tt.expectedField {
					found = true
					break
				}
			}

			s.Require().True(found, "Expected validation error for field '%s', got details: %v", tt.expectedField, errResp.Details)
		})
	}
}
