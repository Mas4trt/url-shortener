//go:build integration
// +build integration

package integration

import (
	"context"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"url-shortener/internal/bootstrap"
	"url-shortener/internal/config"
	"url-shortener/internal/tests/integration/mocks"

	authv1 "github.com/Mas4trt/protos/gen/go/auth/v1"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"google.golang.org/grpc"
)

type IntegrationSuite struct {
	suite.Suite
	ctx            context.Context
	cfg            *config.Config
	server         *httptest.Server
	db             *pgxpool.Pool
	redis          *redis.Client
	pgContainer    testcontainers.Container
	redisContainer testcontainers.Container
	ssoServer      *grpc.Server
	ssoAddr        string
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}

func (s *IntegrationSuite) SetupSuite() {
	s.ctx = context.Background()

	var dbURL string
	var redisAddr string

	wd, _ := os.Getwd()
	absPath := filepath.Join(wd, "..", "..", "..", "migrations")

	// Преобразуем путь для Windows-совместимого формата file://
	// Используем net/url для правильного формирования URI
	migrationURL := (&url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(absPath),
	}).String()

	s.pgContainer, s.db, dbURL =
		startPostgres(s.ctx, s.T())

	s.redisContainer, s.redis, redisAddr =
		startRedis(s.ctx, s.T())

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(s.T(), err)

	s.ssoAddr = listener.Addr().String()

	s.ssoServer = grpc.NewServer()

	authv1.RegisterAuthServer(
		s.ssoServer,
		&mocks.MockSSO{},
	)
	go func() {
		_ = s.ssoServer.Serve(listener)
	}()

	s.cfg = newTestConfig(
		dbURL,
		redisAddr,
		s.ssoAddr,
	)

	require.NoError(
		s.T(),
		bootstrap.RunMigrations(migrationURL, s.cfg),
	)

	handler, cleanup, err :=
		bootstrap.InitializeRouter(s.cfg)

	require.NoError(s.T(), err)

	if cleanup != nil {
		s.T().Cleanup(cleanup)
	}

	s.server = httptest.NewServer(handler)
}

func (s *IntegrationSuite) TearDownSuite() {

	if s.server != nil {
		s.server.Close()
	}

	if s.redis != nil {
		_ = s.redis.Close()
	}

	if s.db != nil {
		s.db.Close()
	}

	if s.pgContainer != nil {
		_ = s.pgContainer.Terminate(s.ctx)
	}

	if s.redisContainer != nil {
		_ = s.redisContainer.Terminate(s.ctx)
	}

	if s.ssoServer != nil {
		s.ssoServer.Stop()
	}
}

func (s *IntegrationSuite) SetupTest() {

	_, err := s.db.Exec(
		s.ctx,
		`TRUNCATE TABLE urlshortener.url RESTART IDENTITY CASCADE`,
	)

	require.NoError(s.T(), err)

	require.NoError(
		s.T(),
		s.redis.FlushAll(s.ctx).Err(),
	)
}
