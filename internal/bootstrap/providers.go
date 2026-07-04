package bootstrap

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"
	"url-shortener/internal/config"
	service "url-shortener/internal/service/url"
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
