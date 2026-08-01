//go:build integration
// +build integration

package integration

import (
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

func (s *IntegrationSuite) TestDeleteURL() {
	tests := []struct {
		name           string
		alias          string
		expectedStatus int
		setup          func()
		validate       func(alias string)
	}{
		{
			name:           "Success - Delete existing alias from DB and Cache",
			alias:          "to-delete",
			expectedStatus: http.StatusNoContent,
			setup: func() {
				_, err := s.db.Exec(
					s.ctx,
					"INSERT INTO urlshortener.url (url, alias) VALUES ($1, $2)",
					"https://example.com/delete-me",
					"to-delete",
				)
				s.Require().NoError(err)

				err = s.redis.Set(s.ctx, "to-delete", "https://example.com/delete-me", 0).Err()
				s.Require().NoError(err)
			},
			validate: func(alias string) {
				var url string
				err := s.db.QueryRow(s.ctx, "SELECT url FROM urlshortener.url WHERE alias = $1", alias).Scan(&url)
				s.Require().ErrorIs(err, pgx.ErrNoRows, "Record MUST be deleted from PostgreSQL")

				err = s.redis.Get(s.ctx, alias).Err()
				s.Require().ErrorIs(err, redis.Nil, "Record MUST be deleted from Redis Cache")
			},
		},
		{
			name:           "Not Found - Delete non-existent alias",
			alias:          "missing-alias",
			expectedStatus: http.StatusNotFound,
			setup:          nil,
			validate:       nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {

			if tt.setup != nil {
				tt.setup()
			}

			req := s.NewRequest(http.MethodDelete, "/"+tt.alias, nil)
			resp := s.Do(req)
			defer resp.Body.Close()

			s.Require().Equal(tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusNotFound {
				var errResp map[string]interface{}
				s.DecodeBody(resp, &errResp)
				s.Require().Equal("Error", errResp["status"])
			}

			if tt.validate != nil {
				tt.validate(tt.alias)
			}
		})
	}
}
