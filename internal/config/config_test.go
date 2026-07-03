package config_test

import (
	"path/filepath"
	"testing"
	"time"
	"url-shortener/internal/config"

	"github.com/stretchr/testify/assert"
)

func assertPanicContains(t *testing.T, f func(), expectedMsg string) {
	t.Helper() // Указывает, что это вспомогательная функция, чтобы ошибки указывали на вызывающий код
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic, but did not panic")
			return
		}
		assert.Contains(t, r, expectedMsg)
	}()
	f()
}

func TestMustLoad(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectPanic bool
		panicMsg    string
		wantCfg     *config.Config
	}{
		{
			name:        "Empty path",
			path:        "",
			expectPanic: true,
			panicMsg:    "config path is empty",
		},
		{
			name:        "File does not exist",
			path:        "non-existent.yaml",
			expectPanic: true,
			panicMsg:    "config file does not exist",
		},
		{
			name:        "Invalid config content",
			path:        filepath.Join("testdata", "invalid.yaml"),
			expectPanic: true,
			panicMsg:    "failed to read config",
		},
		{
			name:        "Valid config",
			path:        filepath.Join("testdata", "valid.yaml"),
			expectPanic: false,
			wantCfg: &config.Config{
				Env:         "local",
				DatabaseURL: "postgres://localhost:5432/test",
				AliasLength: 10,
				MaxRetries:  3,
				ServerConfig: config.HTTPServer{
					Address:     "localhost:8080",
					Timeout:     5 * time.Second,
					IdleTimeout: 30 * time.Second,
				},
			},
		},
		{
			name:        "minimal.yaml",
			path:        filepath.Join("testdata", "minimal.yaml"),
			expectPanic: false,
			wantCfg: &config.Config{
				Env:         "local",
				DatabaseURL: "postgres://localhost:5432/test",
				AliasLength: 6,
				MaxRetries:  5,
				ServerConfig: config.HTTPServer{
					Address:     "localhost:8080",
					Timeout:     4 * time.Second,
					IdleTimeout: 60 * time.Second,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				assertPanicContains(t, func() {
					config.MustLoad(tt.path)
				}, tt.panicMsg)
				return
			}

			gotCfg := config.MustLoad(tt.path)
			assert.Equal(t, tt.wantCfg, gotCfg)
		})
	}
}
