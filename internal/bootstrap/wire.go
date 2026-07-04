package bootstrap

import (
	"net/http"
	"url-shortener/internal/app"
	"url-shortener/internal/config"
	service "url-shortener/internal/service/url"
	dbpostgres "url-shortener/internal/storage/postgres"
	cache "url-shortener/internal/storage/redis"
	httptransport "url-shortener/internal/transport/http"
	"url-shortener/internal/transport/http/handlers"
	"url-shortener/pkg/random"

	"github.com/google/wire"
)

// StorageSet описывает зависимости уровня хранения данных
var StorageSet = wire.NewSet(
	// Слой базы данных (PostgreSQL)
	provideDatabaseURL,
	dbpostgres.InitPostgres,
	wire.Bind(new(cache.Storage), new(*dbpostgres.PostgresRepo)),

	// Слой кэширования (Redis)
	provideRedis,
	provideCacheTTL,
	cache.New,
	wire.Bind(new(service.URLRepository), new(*cache.Cache)),
)

// ServiceSet описывает зависимости слоя бизнес-логики
var ServiceSet = wire.NewSet(
	provideAliasGenerator,
	wire.Bind(new(service.AliasGenerator), new(*random.Generator)),

	provideServiceConfig,
	service.New,
	wire.Bind(new(handlers.URLService), new(*service.Service)),
)

var HTTPSet = wire.NewSet(
	provideValidator,
	handlers.New,
	httptransport.NewRouter,
	httptransport.NewServer,
)

func InitializeRouter(cfg *config.Config) (http.Handler, func(), error) {
	wire.Build(
		provideLogger,
		StorageSet,
		ServiceSet,
		provideValidator,
		handlers.New,
		httptransport.NewRouter,
	)
	return nil, nil, nil
}

// InitializeApp собирает DI-контейнер приложения
func InitializeApp(cfg *config.Config) (*app.App, func(), error) {
	wire.Build(
		provideLogger,
		StorageSet,
		ServiceSet,
		HTTPSet,
		app.New,
	)
	return nil, nil, nil
}
