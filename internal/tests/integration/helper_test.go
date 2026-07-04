package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/stretchr/testify/require"
)

var httpClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func (s *IntegrationSuite) URL(path string) string {
	return s.server.URL + path
}

func (s *IntegrationSuite) NewRequest(
	method string,
	path string,
	body any,
) *http.Request {
	s.T().Helper()

	var reader io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(s.T(), err)

		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(
		method,
		s.URL(path),
		reader,
	)
	require.NoError(s.T(), err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req
}

func (s *IntegrationSuite) Do(req *http.Request) *http.Response {
	s.T().Helper()

	resp, err := httpClient.Do(req)
	require.NoError(s.T(), err)

	return resp
}

func (s *IntegrationSuite) DecodeBody(
	resp *http.Response,
	dst any,
) {
	s.T().Helper()

	defer resp.Body.Close()

	err := json.NewDecoder(resp.Body).Decode(dst)
	require.NoError(s.T(), err)
}

func (s *IntegrationSuite) Body(resp *http.Response) []byte {
	s.T().Helper()

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	return data
}
