package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"
	"url-shortener/internal/authn"
	"url-shortener/internal/config"
	service "url-shortener/internal/service/url"
	"url-shortener/internal/ssoclient"
	cache "url-shortener/internal/storage/redis"
	"url-shortener/internal/transport/http/validation"
	"url-shortener/pkg/random"

	"github.com/go-playground/validator/v10"
	"github.com/golang-migrate/migrate/v4"
	"github.com/redis/go-redis/v9"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"

	defaultSSODialTimeout = 5 * time.Second
)

func provideServiceConfig(cfg *config.Config) service.Config {
	return service.Config{
		MaxRetries: cfg.MaxRetries,
	}
}

func provideAliasGenerator(cfg *config.Config) *random.Generator {
	return random.New(cfg.AliasLength)
}

func provideRedis(cfg *config.Config, log *slog.Logger) (*redis.Client, func(), error) {
	return cache.InitRedis(log, cfg.RedisAddr)
}

func provideDatabaseURL(cfg *config.Config) string {
	return cfg.DatabaseURL
}

func provideCacheTTL(cfg *config.Config) time.Duration {
	return cfg.TTL
}

func provideValidator() *validator.Validate {
	return validation.New()
}

// provideSSOClientOptions fills in a sane default dial timeout since
// config.SSOConfig.DialTimeout is intentionally optional (zero value is
// valid config, not an error).
func provideSSOClientOptions(cfg *config.Config) ssoclient.Options {
	timeout := cfg.SSO.DialTimeout
	if timeout == 0 {
		timeout = defaultSSODialTimeout
	}

	return ssoclient.Options{
		Addr:          cfg.SSO.Addr,
		ApplicationID: cfg.SSO.ApplicationID,
		DialTimeout:   timeout,
	}
}

func provideSSOClient(opts ssoclient.Options) (*ssoclient.Client, func(), error) {
	client, err := ssoclient.New(context.Background(), opts)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to sso: %w", err)
	}

	return client, func() { _ = client.Close() }, nil
}

// provideAuthVerifier builds the local JWT verifier from the secret this
// service was provisioned with in sso (see config.SSOConfig doc comment).
func provideAuthVerifier(cfg *config.Config) *authn.Verifier {
	return authn.NewVerifier(cfg.SSO.AppSecret, cfg.SSO.ApplicationID)
}

func provideLogger(cfg *config.Config) *slog.Logger {
	switch cfg.Env {
	case envLocal:
		return slog.New(
			slog.NewTextHandler(
				os.Stdout,
				&slog.HandlerOptions{
					Level: slog.LevelDebug,
				},
			),
		)

	case envDev:
		return slog.New(
			slog.NewJSONHandler(
				os.Stdout,
				&slog.HandlerOptions{
					Level: slog.LevelDebug,
				},
			),
		)

	case envProd:
		return slog.New(
			slog.NewJSONHandler(
				os.Stdout,
				&slog.HandlerOptions{
					Level: slog.LevelInfo,
				},
			),
		)

	default:
		return slog.New(
			slog.NewJSONHandler(
				os.Stdout,
				&slog.HandlerOptions{
					Level: slog.LevelInfo,
				},
			),
		)
	}
}

func RunMigrations(migrationsPath string, cfg *config.Config) error {
	const op = "bootstrap.RunMigrations"

	m, err := migrate.New(
		migrationsPath,
		cfg.DatabaseURL,
	)
	if err != nil {
		return fmt.Errorf("%s: create migrate instance: %w", op, err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}

		return fmt.Errorf("%s: apply migrations: %w", op, err)
	}

	return nil
}
