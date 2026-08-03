package httptransport

import (
	"log/slog"
	"net/http"
	"time"
	"url-shortener/internal/authn"
	dbpostgres "url-shortener/internal/storage/postgres"
	"url-shortener/internal/transport/http/handlers"
	logger "url-shortener/internal/transport/http/middleware"
	authmw "url-shortener/internal/transport/http/middleware/auth"
	"url-shortener/internal/transport/http/middleware/ratelimit"
	"url-shortener/internal/transport/http/middleware/reqid"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

const limiterEntryTTL = 10 * time.Minute

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
	router.Use(reqid.Propagate) // makes request ID available to service/storage logs too
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)

	authLimiter := ratelimit.New(authEndpointRate, authEndpointBurst, limiterEntryTTL)
	// Auth: proxies to sso, no local auth required to call these.
	router.Route("/auth", func(r chi.Router) {
		r.Use(authLimiter.Middleware)
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)
	})

	// Redirect stays public — anyone with a short link can follow it.
	router.Get("/{alias}", handler.Get)

	// Creating/deleting links requires a valid sso access token, plus a
	// per-IP rate limit since these are the write/expensive paths.
	writeLimiter := ratelimit.New(writeEndpointRate, writeEndpointBurst, limiterEntryTTL)
	router.Group(func(r chi.Router) {
		r.Use(authmw.RequireAuth(verifier))
		r.Use(writeLimiter.Middleware)
		r.Post("/url", handler.Save)
		r.Delete("/{alias}", handler.Delete)
	})

	router.Get("/healthz", handlers.ReadinessProbe(db, rdb))

	return router
}
