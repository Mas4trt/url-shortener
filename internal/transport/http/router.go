package httptransport

import (
	"log/slog"
	"net/http"
	"url-shortener/internal/authn"
	dbpostgres "url-shortener/internal/storage/postgres"
	"url-shortener/internal/transport/http/handlers"
	logger "url-shortener/internal/transport/http/middleware"
	authmw "url-shortener/internal/transport/http/middleware/auth"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

func NewRouter(
	log *slog.Logger,
	handler *handlers.Handler,
	authHandler *handlers.AuthHandler,
	verifier *authn.Verifier,
	db *dbpostgres.PostgresRepo,
	rdb *redis.Client,
) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)

	// Auth: proxies to sso, no local auth required to call these.
	router.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)
	})

	// Redirect stays public — anyone with a short link can follow it.
	router.Get("/{alias}", handler.Get)

	// Creating/deleting links requires a valid sso access token.
	router.Group(func(r chi.Router) {
		r.Use(authmw.RequireAuth(verifier))
		r.Post("/url", handler.Save)
		r.Delete("/{alias}", handler.Delete)
	})

	router.Get("/healthz", handlers.ReadinessProbe(db, rdb))

	return router
}
