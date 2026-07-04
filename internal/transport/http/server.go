package httptransport

import (
	"net/http"
	"url-shortener/internal/config"
)

func NewServer(
	cfg *config.Config,
	handler http.Handler,
) *http.Server {

	return &http.Server{
		Addr:         cfg.ServerConfig.Address,
		Handler:      handler,
		ReadTimeout:  cfg.ServerConfig.Timeout,
		WriteTimeout: cfg.ServerConfig.Timeout,
		IdleTimeout:  cfg.ServerConfig.IdleTimeout,
	}
}
