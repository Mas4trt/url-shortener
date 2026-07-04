package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/config" // Замените на ваш актуальный модуль
)

// createTempConfig создает временный yaml-файл для тестов
func createTempConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "config.yaml")

	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err, "failed to create temp config file")

	return filePath
}

func TestMustLoad(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		configPath  string
		envVars     map[string]string
		expectPanic bool
		panicMsg    string
		wantCfg     *config.Config
	}{
		{
			name:        "Empty path",
			configPath:  "",
			expectPanic: true,
			panicMsg:    "config path is empty",
		},
		{
			name:        "File does not exist",
			configPath:  "non-existent-file-path.yaml",
			expectPanic: true,
			panicMsg:    "config file does not exist",
		},
		{
			name:        "Invalid config content (bad yaml)",
			yamlContent: `invalid: yaml: content: -`,
			expectPanic: true,
			panicMsg:    "failed to read config",
		},
		{
			name: "Missing required fields",
			yamlContent: `
alias_length: 10
http_server:
  address: "localhost:8080"
`,
			expectPanic: true,
			panicMsg:    "failed to read config",
		},
		{
			name: "Valid full config",
			yamlContent: `
env: "prod"
database_url: "postgres://user:pass@localhost:5432/db"
redis_url: "localhost:6379"  
alias_length: 8
max_retries: 3
http_server:
  address: "127.0.0.1:9090"
  timeout: 10s
  idle_timeout: 120s
`,
			expectPanic: false,
			wantCfg: &config.Config{
				Env:         "prod",
				DatabaseURL: "postgres://user:pass@localhost:5432/db",
				RedisAddr:   "localhost:6379",
				AliasLength: 8,
				MaxRetries:  3,
				TTL:         3600 * time.Second,
				ServerConfig: config.HTTPServer{
					Address:     "127.0.0.1:9090",
					Timeout:     10 * time.Second,
					IdleTimeout: 120 * time.Second,
				},
			},
		},
		{
			name: "Minimal config (testing defaults)",
			yamlContent: `
env: "local"
database_url: "postgres://localhost:5432/test"
redis_url: "localhost:6379" # Добавьте сюда
`,
			expectPanic: false,
			wantCfg: &config.Config{
				Env:         "local",
				DatabaseURL: "postgres://localhost:5432/test",
				RedisAddr:   "localhost:6379",
				AliasLength: 6,
				MaxRetries:  5,
				TTL:         3600 * time.Second,
				ServerConfig: config.HTTPServer{
					Address:     "localhost:8080",
					Timeout:     4 * time.Second,
					IdleTimeout: 60 * time.Second,
				},
			},
		},
		{
			name: "Environment variables override",
			yamlContent: `
env: "local"
database_url: "postgres://old:old@localhost:5432/db"
redis_url: "localhost:6379"  # ДОБАВЬТЕ ЭТО
alias_length: 6
`,
			envVars: map[string]string{
				"DATABASE_URL": "postgres://new:new@remote:5432/db",
			},
			expectPanic: false,
			wantCfg: &config.Config{
				Env:         "local",
				DatabaseURL: "postgres://new:new@remote:5432/db",
				RedisAddr:   "localhost:6379", // ДОБАВЬТЕ ЭТО
				AliasLength: 6,
				MaxRetries:  5,
				TTL:         3600 * time.Second, // И это
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
			if tt.envVars != nil {
				for k, v := range tt.envVars {
					t.Setenv(k, v)
				}
			}

			var path string
			if tt.yamlContent != "" {
				path = createTempConfig(t, tt.yamlContent)
			} else {
				path = tt.configPath
			}

			if tt.expectPanic {
				assertPanicContains(t, func() {
					config.MustLoad(path)
				}, tt.panicMsg)
				return
			}

			gotCfg := config.MustLoad(path)
			assert.Equal(t, tt.wantCfg, gotCfg)
		})
	}
}

// assertPanicContains проверяет, что функция паникует и сообщение паники содержит ожидаемый текст
func assertPanicContains(t *testing.T, f func(), expectedMsg string) {
	t.Helper()
	defer func() {
		r := recover()
		require.NotNil(t, r, "expected panic, but did not panic")

		errStr, ok := r.(string)
		if !ok {
			errStr = r.(error).Error()
		}

		assert.True(t, strings.Contains(errStr, expectedMsg),
			"panic message %q does not contain %q", errStr, expectedMsg)
	}()
	f()
}
