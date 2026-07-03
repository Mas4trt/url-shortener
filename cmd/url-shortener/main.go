package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"url-shortener/internal/config"
	service "url-shortener/internal/service/url"
	"url-shortener/internal/storage/postgres"
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
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	configPath := fetchConfigPath()
	cfg := config.MustLoad(configPath)

	log := setupLogger(cfg.Env)
	log.Info("starting url-shortener", slog.String("env", cfg.Env))

	runMigrations(cfg.DatabaseURL, log)

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCancel()

	storage, err := postgres.New(dbCtx, cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}
	defer storage.Close()

	log.Info("storage initialized successfully")

	svcCfg := service.Config{
		AliasLength: cfg.AliasLength,
		MaxRetries:  cfg.MaxRetries,
	}

	aliasGen := random.New()

	urlService, err := service.New(log, storage, aliasGen, svcCfg)
	if err != nil {
		log.Error("failed to init urlService", sl.Err(err))
		os.Exit(1)
	}

	val := validator.New()
	urlHandler := handlers.New(log, urlService, val)

	// Настройка роутера
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)

	// Роуты
	router.Post("/url", urlHandler.Save)
	router.Get("/{alias}", urlHandler.Get)
	router.Delete("/{alias}", urlHandler.Delete)

	// Конфигурация HTTP-сервера
	srv := &http.Server{
		Addr:         cfg.ServerConfig.Address,
		Handler:      router,
		ReadTimeout:  cfg.ServerConfig.Timeout,
		WriteTimeout: cfg.ServerConfig.Timeout,
		IdleTimeout:  cfg.ServerConfig.IdleTimeout,
	}

	// Канал для отслеживания системных сигналов завершения
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("starting HTTP server", slog.String("address", cfg.ServerConfig.Address))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed to start server", sl.Err(err))
		}
	}()

	<-done
	log.Info("stopping server gracefully...")

	// Даем серверу 10 секунд на завершение текущих запросов
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("failed to shutdown server gracefully", sl.Err(err))
	}

	log.Error("server stopped completely")
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

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
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
