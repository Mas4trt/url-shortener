package httptransport

import (
	"log/slog"
	"net/http"
	dbpostgres "url-shortener/internal/storage/postgres"
	"url-shortener/internal/transport/http/handlers"
	logger "url-shortener/internal/transport/http/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

func NewRouter(
	log *slog.Logger,
	handler *handlers.Handler,
	db *dbpostgres.PostgresRepo,
	rdb *redis.Client,
) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)

	router.Post("/url", handler.Save)
	router.Get("/{alias}", handler.Get)
	router.Delete("/{alias}", handler.Delete)

	router.Get("/healthz", handlers.ReadinessProbe(db, rdb))

	return router
}
