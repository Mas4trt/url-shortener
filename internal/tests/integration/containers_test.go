//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"time"

	"url-shortener/internal/config"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	postgresUser     = "postgres"
	postgresPassword = "postgres"
	postgresDB       = "urlshortener"

	redisPort = "6379"

	testAppSecret = "integration-test-app-secret-dont-use-this-in-production"
)

func startPostgres(
	ctx context.Context,
	t require.TestingT,
) (testcontainers.Container, *pgxpool.Pool, string) {

	container, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(postgresDB),
		postgres.WithUsername(postgresUser),
		postgres.WithPassword(postgresPassword),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp"),
		),
	)

	require.NoError(t, err)

	connString, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connString)
	require.NoError(t, err)

	require.NoError(t, pool.Ping(ctx))

	return container, pool, connString
}

func startRedis(
	ctx context.Context,
	t require.TestingT,
) (testcontainers.Container, *redis.Client, string) {

	container, err := tcredis.Run(
		ctx,
		"redis:7-alpine",
	)

	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, redisPort)
	require.NoError(t, err)

	addr := fmt.Sprintf("%s:%s", host, port.Port())

	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	require.NoError(t, client.Ping(ctx).Err())

	return container, client, addr
}

func newTestConfig(
	dbURL string,
	redisAddr string,
	ssoAddr string,
) *config.Config {

	return &config.Config{
		Env: "test",

		DatabaseURL: dbURL,

		RedisAddr: redisAddr,

		MaxRetries: 5,

		AliasLength: 8,

		TTL: time.Minute,

		SSO: config.SSOConfig{
			Addr:          ssoAddr,
			ApplicationID: 1,
			AppSecret:     testAppSecret,
			DialTimeout:   5 * time.Second,
		},

		ServerConfig: config.HTTPServer{
			Address:     ":0",
			Timeout:     5 * time.Second,
			IdleTimeout: 30 * time.Second,
		},
	}
}
