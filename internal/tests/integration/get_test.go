//go:build integration
// +build integration

package integration

import (
	"net/http"
	"time"
)

func (s *IntegrationSuite) TestGetURL() {
	tests := []struct {
		name           string
		alias          string
		expectedStatus int
		expectedLoc    string
		setup          func()
	}{
		{
			name:           "Success - Valid Alias from Database",
			alias:          "gopher",
			expectedStatus: http.StatusFound,
			expectedLoc:    "https://go.dev",
			setup: func() {
				_, err := s.db.Exec(
					s.ctx,
					"INSERT INTO urlshortener.url (url, alias) VALUES ($1, $2)",
					"https://go.dev",
					"gopher",
				)
				s.Require().NoError(err)
			},
		},
		{
			name:           "Success - Valid Alias from Cache (Redis)",
			alias:          "fast-redis",
			expectedStatus: http.StatusFound,
			expectedLoc:    "https://redis.io",
			setup: func() {
				err := s.redis.Set(s.ctx, "fast-redis", "https://redis.io", time.Minute).Err()
				s.Require().NoError(err)
			},
		},
		{
			name:           "Not Found - Non-existent Alias",
			alias:          "missing-alias",
			expectedStatus: http.StatusNotFound,
			expectedLoc:    "",
			setup:          nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {

			if tt.setup != nil {
				tt.setup()
			}

			req := s.NewRequest(http.MethodGet, "/"+tt.alias, nil)
			resp := s.Do(req)
			defer resp.Body.Close()

			s.Require().Equal(tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusFound {
				loc := resp.Header.Get("Location")
				s.Require().Equal(tt.expectedLoc, loc, "Location header should match target URL")
			} else {
				var errResp map[string]interface{}
				s.DecodeBody(resp, &errResp)
				s.Require().Equal("Error", errResp["status"], "Response status field should indicate Error")
			}
		})
	}
}
