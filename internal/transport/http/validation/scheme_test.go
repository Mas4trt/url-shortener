package validation_test

import (
	"testing"

	"url-shortener/internal/transport/http/validation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type urlSchemeFixture struct {
	URL string `validate:"url_scheme"`
}

func TestURLScheme(t *testing.T) {
	v, err := validation.New()
	require.NoError(t, err)

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"http allowed", "http://example.com", false},
		{"https allowed", "https://example.com/path?query=1", false},
		{"javascript scheme rejected", "javascript:alert(1)", true},
		{"data scheme rejected", "data:text/html,<script>alert(1)</script>", true},
		{"file scheme rejected", "file:///etc/passwd", true},
		{"schemeless url rejected", "example.com/path", true},
		{"empty string rejected", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(urlSchemeFixture{URL: tt.url})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
