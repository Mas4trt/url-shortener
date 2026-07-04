package main

import (
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"url-shortener/internal/config"
	service "url-shortener/internal/service/url"
	cache "url-shortener/internal/storage/redis"
	"url-shortener/internal/transport/http/handlers"
	logger "url-shortener/internal/transport/http/middleware"
	sl "url-shortener/pkg/logger/sl"
	"url-shortener/pkg/random"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/redis/go-redis/v9"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad(fetchConfigPath())

	log := setupLogger(cfg)

	runMigrations(cfg.DatabaseURL, log)

	app, err := InitializeApp(cfg)
	if err != nil {
		panic(err)
	}

	if err := app.Run(); err != nil {
		panic(err)
	}
}

// Запускает миграции при старте приложения
func runMigrations(dbURL string, log *slog.Logger) {
	path := "migrations"
	m, err := migrate.New("file://"+path, dbURL)
	if err != nil {
		log.Error("failed to initialize migrations", sl.Err(err))
		os.Exit(1)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("no new migrations to apply")
			return
		}
		log.Error("failed to apply migrations", sl.Err(err))
		os.Exit(1)
	}
	log.Info("migrations applied successfully")
}

// fetchConfigPath выбирает откуда взять путь к конфигу
// Приоритет: флаг командной строки -> переменная окружения
func fetchConfigPath() string {
	var res string

	// flag.StringVar позволяет передавать --config="path/to/config.yaml"
	// Проверка flag.Parsed() нужна, чтобы не вызывать Parse повторно в тестах
	if !flag.Parsed() {
		flag.StringVar(&res, "config", "", "path to configuration file")
		flag.Parse()
	}

	// Если флаг пустой, смотрим в окружение
	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}

func setupLogger(cfg *config.Config) *slog.Logger {
	var log *slog.Logger

	switch cfg.Env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func provideRouter(log *slog.Logger, urlHandler *handlers.Handler) chi.Router {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)

	// Роуты
	router.Post("/url", urlHandler.Save)
	router.Get("/{alias}", urlHandler.Get)
	router.Delete("/{alias}", urlHandler.Delete)

	return router
}

func provideHTTPServer(cfg *config.Config, router chi.Router) *http.Server {
	return &http.Server{
		Addr:         cfg.ServerConfig.Address,
		Handler:      router,
		ReadTimeout:  cfg.ServerConfig.Timeout,
		WriteTimeout: cfg.ServerConfig.Timeout,
		IdleTimeout:  cfg.ServerConfig.IdleTimeout,
	}
}

func provideServiceConfig(cfg *config.Config) service.Config {
	return service.Config{
		MaxRetries: cfg.MaxRetries,
	}
}

func provideAliasGenerator(cfg *config.Config) *random.Generator {
	return random.New(cfg.AliasLength)
}

func provideRedis(cfg *config.Config, log *slog.Logger) (*redis.Client, error) {
	return cache.InitRedis(log, cfg.RedisAddr)
}

func provideDatabaseURL(cfg *config.Config) string {
	return cfg.DatabaseURL
}

func provideCacheTTL(cfg *config.Config) time.Duration {
	return cfg.TTL
}

func provideValidator() *validator.Validate {
	return validator.New()
}
