//go:build wireinject
// +build wireinject

package main

import (
	"url-shortener/internal/app"
	"url-shortener/internal/config"
	service "url-shortener/internal/service/url"
	dbpostgres "url-shortener/internal/storage/postgres"
	cache "url-shortener/internal/storage/redis"
	"url-shortener/internal/transport/http/handlers"
	"url-shortener/pkg/random"

	"github.com/google/wire"
)

var StorageSet = wire.NewSet(
	provideDatabaseURL,
	dbpostgres.InitPostgres,

	wire.Bind(new(service.URLRepository), new(*cache.Cache)),
	wire.Bind(new(cache.Storage), new(*dbpostgres.PostgresRepo)),

	provideCacheTTL,
	provideRedis,
	cache.New,
)

var ServiceSet = wire.NewSet(
	provideAliasGenerator,
	provideServiceConfig,
	service.New,

	wire.Bind(
		new(handlers.URLService),
		new(*service.Service),
	),
	wire.Bind(
		new(service.AliasGenerator),
		new(*random.Generator),
	),
)

var HTTPSet = wire.NewSet(
	provideValidator,
	handlers.New,
	provideRouter,
	provideHTTPServer,
)

func InitializeApp(cfg *config.Config) (*app.App, error) {
	wire.Build(
		setupLogger,
		StorageSet,
		ServiceSet,
		HTTPSet,
		app.New,
	)
	return &app.App{}, nil
}
