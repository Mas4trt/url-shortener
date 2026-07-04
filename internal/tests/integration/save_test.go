package integration

import (
	"net/http"
	"url-shortener/internal/transport/http/handlers"
)

func (s *IntegrationSuite) TestSaveURL() {
	// Table-Driven Tests структура
	tests := []struct {
		name           string
		reqBody        map[string]string
		expectedStatus int
		setup          func()
		validate       func(alias string)
	}{
		{
			name: "Success - Auto-generated Alias",
			reqBody: map[string]string{
				"url": "https://example.com/very/long/and/complex/path",
			},
			expectedStatus: http.StatusOK,
			validate: func(alias string) {
				s.Require().NotEmpty(alias, "Alias must not be empty")
				s.Require().Len(alias, s.cfg.AliasLength, "Alias length should match config")

				var dbURL string
				err := s.db.QueryRow(s.ctx, "SELECT url FROM urlshortener.url WHERE alias = $1", alias).Scan(&dbURL)
				s.Require().NoError(err, "Record must exist in the database")
				s.Require().Equal("https://example.com/very/long/and/complex/path", dbURL)
			},
		},
		{
			name: "Success - Custom Alias",
			reqBody: map[string]string{
				"url":   "https://github.com",
				"alias": "my-github",
			},
			expectedStatus: http.StatusOK,
			validate: func(alias string) {
				s.Require().Equal("my-github", alias)

				var dbURL string
				err := s.db.QueryRow(s.ctx, "SELECT url FROM urlshortener.url WHERE alias = $1", alias).Scan(&dbURL)
				s.Require().NoError(err)
				s.Require().Equal("https://github.com", dbURL)
			},
		},
		{
			name: "Conflict - Alias Already Exists",
			reqBody: map[string]string{
				"url":   "https://golang.org",
				"alias": "duplicate-alias",
			},
			setup: func() {
				_, err := s.db.Exec(s.ctx, "INSERT INTO urlshortener.url (url, alias) VALUES ($1, $2)", "https://old.com", "duplicate-alias")
				s.Require().NoError(err)
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "Bad Request - Invalid URL Format",
			reqBody: map[string]string{
				"url": "not-a-valid-url-format",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Bad Request - Empty URL",
			reqBody: map[string]string{
				"url": "",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.setup != nil {
				tt.setup()
			}

			req := s.NewRequest(http.MethodPost, "/url", tt.reqBody)
			resp := s.Do(req)

			s.Require().Equal(tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var respBody handlers.SaveResponse
				s.DecodeBody(resp, &respBody)

				s.Require().Equal("OK", respBody.Status)

				if tt.validate != nil {
					tt.validate(respBody.Alias)
				}
			} else {
				var errResp map[string]interface{}
				s.DecodeBody(resp, &errResp)
				s.Require().Equal("Error", errResp["status"])
			}
		})
	}
}
