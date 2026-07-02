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
	"url-shortener/internal/delivery/http/handlers"
	"url-shortener/internal/delivery/http/middleware/logger"
	sl "url-shortener/internal/lib/logger/sl"
	service "url-shortener/internal/service/url"
	"url-shortener/internal/storage/sqlite"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	//TODO: init config: cleanenv
	configPath := fetchConfigPath()
	cfg := config.MustLoad(configPath)

	//TODO: init logger: slog
	log := setupLogger(cfg.Env)
	log.Info("starting url-shortener", slog.String("env", cfg.Env))

	//TODO: init storage: sqlite
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCancel()

	storage, err := sqlite.New(dbCtx, cfg.StoragePath)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	defer func() {
		if err := storage.Close(); err != nil {
			log.Error("failed to close storage", sl.Err(err))
		}
	}()
	log.Info("storage initialized successfully")

	//TODO: run server
	urlService := service.New(log, storage)
	val := validator.New()
	urlHandler := handlers.NewURLHandler(log, urlService, val)

	//TODO: init router: chi, chi render
	// Настройка роутера
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)

	// Роуты
	router.Post("/url", urlHandler.Save)

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

	// Закрываем базу данных после остановки сервера
	if err := storage.Close(); err != nil {
		log.Error("failed to close storage", sl.Err(err))
	}

	log.Error("server stopped completely")
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
