package integration

import (
	"context"
	"fmt"
	"time"

	"url-shortener/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

const (
	postgresUser     = "postgres"
	postgresPassword = "postgres"
	postgresDB       = "urlshortener"

	redisPort = "6379"
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
) *config.Config {

	return &config.Config{
		Env: "test",

		DatabaseURL: dbURL,

		RedisAddr: redisAddr,

		MaxRetries: 5,

		AliasLength: 8,

		TTL: time.Minute,

		ServerConfig: config.HTTPServer{
			Address:     ":0",
			Timeout:     5 * time.Second,
			IdleTimeout: 30 * time.Second,
		},
	}
}
